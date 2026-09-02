package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/agentconfig"
	"github.com/mlko01/lyrahcle/apps/backend/internal/alerts"
	"github.com/mlko01/lyrahcle/apps/backend/internal/bootstrap"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/db"
	"github.com/mlko01/lyrahcle/apps/backend/internal/dbscope"
	"github.com/mlko01/lyrahcle/apps/backend/internal/joblock"
	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/observability"
	"github.com/mlko01/lyrahcle/apps/backend/internal/redisclient"
	"github.com/mlko01/lyrahcle/apps/backend/internal/scheduler"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	shutdownTracing, err := observability.Init(ctx, cfg.OTLPTraceEndpoint)
	if err != nil {
		log.Fatalf("tracing init: %v", err)
	}
	defer shutdownTracing(context.Background())

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	log.Println("migrations applied")

	stores := bootstrap.NewStores(pool)

	agentConfigSigner, agentConfigKeyProvider, err := bootstrap.BuildAgentConfigSigner(cfg)
	if err != nil {
		log.Fatalf("agent config signing configuration: %v", err)
	}

	if err := seedAdmin(ctx, stores.Users, cfg); err != nil {
		log.Printf("seed admin: %v", err)
	}

	redisClient, err := bootstrap.ConnectRedis(cfg)
	if err != nil {
		if cfg.IsProduction() {
			log.Fatalf("redis connect: %v", err)
		}
		log.Printf("WARNING: Redis not available at %s (%v). Rate limiting will use an in-memory counter, and token revocation and response caching will be disabled.", redisclient.SafeTarget(cfg.RedisURL), err)
	} else {
		log.Printf("connected to Redis at %s", redisclient.SafeTarget(cfg.RedisURL))
		defer redisClient.Close()
	}

	alertsCtx, stopAlertsWorker := context.WithCancel(context.Background())
	defer stopAlertsWorker()
	go alerts.NewWorker(redisClient, stores.SpaceCVEs, stores.Assets, stores.AssetAlerts, stores.AuditLog).Run(alertsCtx)

	jobLocker := joblock.New(redisClient)
	sched := scheduler.New()
	if err := sched.Every("0 */6 * * *", "recon-sweep", func() {
		runSingletonScheduledJob(context.Background(), jobLocker, "recon-sweep", 30*time.Minute, func() {
			log.Println("recon sweep tick (not yet implemented)")
		})
	}); err != nil {
		log.Fatalf("scheduler: register recon-sweep: %v", err)
	}
	if err := sched.Every("0 */6 * * *", "tle-sync", func() {
		runSingletonScheduledJob(context.Background(), jobLocker, "tle-sync", 30*time.Minute, func() {
			syncTLEs(ctx, cfg.CelestrakGroups, stores.Satellites, tlePolicy(cfg))
		})
	}); err != nil {
		log.Fatalf("scheduler: register tle-sync: %v", err)
	}
	if _, ok := agentConfigKeyProvider.(keyprovider.LifecycleProvider); ok {
		if err := sched.Every("17 * * * *", "signing-key-rotation-check", func() {
			runSingletonScheduledJob(context.Background(), jobLocker, "signing-key-rotation-check", 10*time.Minute, func() {
				if err := runSigningKeyRotationCheck(context.Background(), agentConfigKeyProvider, stores.AuditLog, time.Now().UTC()); err != nil {
					log.Printf("signing-key rotation check failed: %v", err)
				}
			})
		}); err != nil {
			log.Fatalf("scheduler: register signing-key-rotation-check: %v", err)
		}
	}
	sched.Start()
	defer sched.Stop()

	go runSingletonScheduledJob(context.Background(), jobLocker, "tle-sync", 30*time.Minute, func() {
		syncTLEs(ctx, cfg.CelestrakGroups, stores.Satellites, tlePolicy(cfg))
	})

	runSingletonScheduledJob(ctx, jobLocker, "signing-key-rotation-check", 10*time.Minute, func() {
		if err := runSigningKeyRotationCheck(ctx, agentConfigKeyProvider, stores.AuditLog, time.Now().UTC()); err != nil {
			log.Printf("initial signing-key rotation check failed: %v", err)
		}
	})

	r := newRouterWithBootstrap(cfg, pool, stores, redisClient, agentConfigSigner, agentConfigKeyProvider)

	srv := bootstrap.NewHTTPServer(cfg, r)
	if err := bootstrap.ConfigureAgentClientTLS(srv, cfg); err != nil {
		log.Fatalf("agent mTLS configuration: %v", err)
	}

	serveErr := make(chan error, 1)
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		log.Printf("lyrahcle backend listening on :%s (TLS enabled)", cfg.Port)
		go func() {
			serveErr <- srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		}()
	} else {
		log.Printf("lyrahcle backend listening on :%s (plain HTTP)", cfg.Port)
		go func() {
			serveErr <- srv.ListenAndServe()
		}()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	case sig := <-stop:
		log.Printf("received %s, shutting down gracefully (draining in-flight requests)...", sig)

		stopAlertsWorker()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown timed out, forcing close: %v", err)
			if closeErr := srv.Close(); closeErr != nil {
				log.Printf("forced server close failed: %v", closeErr)
			}
		}

		log.Println("shutdown complete")
	}
}

func newRouter(
	cfg config.Config,
	pool *pgxpool.Pool,
	assets *store.AssetStore,
	programs *store.ProgramStore,
	rf *store.RFStore,
	rfAnomalies *store.AnomalyStore,
	satellites *store.SatelliteStore,
	users *store.UserStore,
	agentKeys *store.AgentKeyStore,
	auditLog *store.AuditStore,
	spaceCVEs *store.SpaceCVEStore,
	assetAlerts *store.AssetAlertStore,
	incidents *store.IncidentStore,
	rbac *store.RBACStore,
	redisClient *redis.Client,
) http.Handler {
	stores := bootstrap.NewStores(pool)
	stores.Assets = assets
	stores.Programs = programs
	stores.RF = rf
	stores.RFAnomalies = rfAnomalies
	stores.Satellites = satellites
	stores.Users = users
	stores.AgentKeys = agentKeys
	stores.AuditLog = auditLog
	stores.SpaceCVEs = spaceCVEs
	stores.AssetAlerts = assetAlerts
	stores.Incidents = incidents
	stores.RBAC = rbac
	agentConfigSigner, agentConfigKeyProvider, err := bootstrap.BuildAgentConfigSigner(cfg)
	if err != nil {
		panic(fmt.Sprintf("agent config signing configuration: %v", err))
	}
	return newRouterWithBootstrap(cfg, pool, stores, redisClient, agentConfigSigner, agentConfigKeyProvider)
}

func newRouterWithBootstrap(
	cfg config.Config,
	pool *pgxpool.Pool,
	stores bootstrap.Stores,
	redisClient *redis.Client,
	agentConfigSigner *agentconfig.Signer,
	agentConfigKeyProvider keyprovider.Provider,
) http.Handler {
	agentCA, err := bootstrap.BuildAgentCA(cfg)
	if err != nil {
		panic(fmt.Sprintf("agent CA configuration: %v", err))
	}
	services, err := bootstrap.BuildRequestServices(cfg, stores.Users, redisClient)
	if err != nil {
		panic(fmt.Sprintf("request services configuration: %v", err))
	}
	tenantLimiter := services.RateLimiters.Tenant

	r := chi.NewRouter()
	r.Use(dbscope.Middleware)
	r.Use(sanitizeRequestID)
	r.Use(middleware.RequestID)
	r.Use(requestIDResponseHeader)
	r.Use(structuredRequestLogger)
	r.Use(observability.Middleware)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders(cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""))
	r.Use(metrics.Middleware)
	r.Use(maxBodySize(1 << 20))
	r.Use(strictCORS(cfg.CORSAllowedOrigins))
	r.Use(services.RateLimiters.Global)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"lyrahcle-backend"}`))
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		ready := pool != nil && pool.Ping(ctx) == nil
		if ready && redisClient != nil {
			ready = redisClient.Ping(ctx).Err() == nil
		}
		w.Header().Set("Content-Type", "application/json")
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not_ready"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/docs", serveAPIDocs)
	r.Handle("/metrics", promhttp.Handler())

	api := &apiRoutes{
		cfg:                    cfg,
		assets:                 stores.Assets,
		programs:               stores.Programs,
		rf:                     stores.RF,
		rfAnomalies:            stores.RFAnomalies,
		satellites:             stores.Satellites,
		users:                  stores.Users,
		agentKeys:              stores.AgentKeys,
		auditLog:               stores.AuditLog,
		spaceCVEs:              stores.SpaceCVEs,
		assetAlerts:            stores.AssetAlerts,
		incidents:              stores.Incidents,
		rbac:                   stores.RBAC,
		redisClient:            redisClient,
		agentConfigSigner:      agentConfigSigner,
		agentConfigKeyProvider: agentConfigKeyProvider,
		detectionRules:         stores.DetectionRules,
		blocklist:              services.Blocklist,
		sessions:               services.Sessions,
		cache:                  services.Cache,
		webAuthnCredentials:    stores.WebAuthnCredentials,
		passwordResets:         stores.PasswordResets,
		searchStore:            stores.Search,
		agentIdentities:        stores.AgentIdentities,
		agentConfigs:           stores.AgentConfigs,
		fleetRollouts:          stores.FleetRollouts,
		orbitalEvents:          stores.OrbitalEvents,
		organizations:          stores.Organizations,
		tenantScope:            stores.TenantScope,
		agentCA:                agentCA,
		webAuthnFlows:          services.WebAuthnFlows,
		webAuthnHandler:        services.WebAuthnHandler,
		mailClient:             services.MailClient,
		isTokenRevoked:         services.Revocation,
		loginLimiter:           services.RateLimiters.Login,
		tenantLimiter:          tenantLimiter,
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authenticatedTenantRateLimit(cfg.JWTSecret, cfg.JWTPreviousSecret, tenantLimiter))
		api.register(r)
	})

	return r
}
