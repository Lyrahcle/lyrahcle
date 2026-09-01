package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/agentconfig"
	"github.com/mlko01/lyrahcle/apps/backend/internal/alerts"
	"github.com/mlko01/lyrahcle/apps/backend/internal/anomaly"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/celestrak"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/db"
	"github.com/mlko01/lyrahcle/apps/backend/internal/dbscope"
	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
	"github.com/mlko01/lyrahcle/apps/backend/internal/joblock"
	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/mailer"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/observability"
	"github.com/mlko01/lyrahcle/apps/backend/internal/orbit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/redisclient"
	"github.com/mlko01/lyrahcle/apps/backend/internal/respcache"
	"github.com/mlko01/lyrahcle/apps/backend/internal/scheduler"
	"github.com/mlko01/lyrahcle/apps/backend/internal/sessionstore"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tlequality"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tokenblocklist"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"github.com/mlko01/lyrahcle/apps/backend/internal/webauthnflow"
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

	assets := store.NewAssetStore(pool)
	programs := store.NewProgramStore(pool)
	rf := store.NewRFStore(pool)
	rfAnomalies := store.NewAnomalyStore(pool)
	satellites := store.NewSatelliteStore(pool)
	users := store.NewUserStore(pool)
	agentKeys := store.NewAgentKeyStore(pool)
	auditLog := store.NewAuditStore(pool)
	spaceCVEs := store.NewSpaceCVEStore(pool)
	assetAlerts := store.NewAssetAlertStore(pool)
	incidents := store.NewIncidentStore(pool)
	rbac := store.NewRBACStore(pool)

	agentConfigSigner, agentConfigKeyProvider, err := buildAgentConfigSigner(cfg)
	if err != nil {
		log.Fatalf("agent config signing configuration: %v", err)
	}

	if err := seedAdmin(ctx, users, cfg); err != nil {
		log.Printf("seed admin: %v", err)
	}

	redisClient, err := redisclient.Connect(cfg.RedisURL, 5*time.Second)
	if err != nil {
		if cfg.IsProduction() {
			log.Fatalf("redis connect: %v", err)
		}
		log.Printf("WARNING: Redis not available at %s (%v). Rate limiting will use an in-memory counter, and token revocation and response caching will be disabled.", cfg.RedisURL, err)
	} else {
		log.Printf("connected to Redis at %s", cfg.RedisURL)
		defer redisClient.Close()
	}

	alertsCtx, stopAlertsWorker := context.WithCancel(context.Background())
	defer stopAlertsWorker()
	go alerts.NewWorker(redisClient, spaceCVEs, assets, assetAlerts, auditLog).Run(alertsCtx)

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
			syncTLEs(ctx, cfg.CelestrakGroups, satellites, tlePolicy(cfg))
		})
	}); err != nil {
		log.Fatalf("scheduler: register tle-sync: %v", err)
	}
	if _, ok := agentConfigKeyProvider.(keyprovider.LifecycleProvider); ok {
		if err := sched.Every("17 * * * *", "signing-key-rotation-check", func() {
			runSingletonScheduledJob(context.Background(), jobLocker, "signing-key-rotation-check", 10*time.Minute, func() {
				if err := runSigningKeyRotationCheck(context.Background(), agentConfigKeyProvider, auditLog, time.Now().UTC()); err != nil {
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
		syncTLEs(ctx, cfg.CelestrakGroups, satellites, tlePolicy(cfg))
	})

	runSingletonScheduledJob(ctx, jobLocker, "signing-key-rotation-check", 10*time.Minute, func() {
		if err := runSigningKeyRotationCheck(ctx, agentConfigKeyProvider, auditLog, time.Now().UTC()); err != nil {
			log.Printf("initial signing-key rotation check failed: %v", err)
		}
	})

	r := newRouterWithAgentConfigSigning(cfg, pool, assets, programs, rf, rfAnomalies, satellites, users, agentKeys, auditLog, spaceCVEs, assetAlerts, incidents, rbac, redisClient, agentConfigSigner, agentConfigKeyProvider)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := configureAgentClientTLS(srv, cfg); err != nil {
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
	agentConfigSigner, agentConfigKeyProvider, err := buildAgentConfigSigner(cfg)
	if err != nil {
		panic(fmt.Sprintf("agent config signing configuration: %v", err))
	}
	return newRouterWithAgentConfigSigning(cfg, pool, assets, programs, rf, rfAnomalies, satellites, users, agentKeys, auditLog, spaceCVEs, assetAlerts, incidents, rbac, redisClient, agentConfigSigner, agentConfigKeyProvider)
}

func newRouterWithAgentConfigSigning(
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
	agentConfigSigner *agentconfig.Signer,
	agentConfigKeyProvider keyprovider.Provider,
) http.Handler {
	detectionRules := store.NewDetectionRuleStore(pool)
	blocklist := tokenblocklist.New(redisClient)
	sessions := sessionstore.New(redisClient)
	cache := respcache.New(redisClient)
	webAuthnCredentials := store.NewWebAuthnStore(pool)
	passwordResets := store.NewPasswordResetStore(pool)
	searchStore := store.NewSearchStore(pool)
	agentIdentities := store.NewAgentIdentityStore(pool)
	agentConfigs := store.NewAgentConfigStore(pool)
	fleetRollouts := store.NewFleetRolloutStore(pool)
	orbitalEvents := store.NewOrbitalEventStore(pool)
	organizations := store.NewOrganizationStore(pool)
	tenantScope := store.NewTenantScopeStore(pool)
	agentCA, err := buildAgentCA(cfg)
	if err != nil {
		panic(fmt.Sprintf("agent CA configuration: %v", err))
	}
	webAuthnFlows := webauthnflow.New(redisClient)
	webAuthnHandler := buildWebAuthnHandler(cfg)
	mailClient := mailer.New(mailer.Config{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, ServerName: cfg.SMTPServerName})
	isTokenRevoked := func(ctx context.Context, claims *auth.Claims) (bool, error) {
		user, err := users.GetByID(ctx, claims.UserID)
		if err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				return true, nil
			}
			return false, err
		}
		if !user.Enabled || user.Role != string(claims.Role) || user.AuthVersion != claims.AuthVersion {
			return true, nil
		}
		if claims.OrganizationID != uuid.Nil && user.OrganizationID != claims.OrganizationID {
			return true, nil
		}
		if user.MustChangePassword != claims.PasswordChangeRequired {
			return true, nil
		}
		issuedAt := time.Unix(0, claims.SessionIssuedAt).UTC()
		return blocklist.Check(ctx, claims.ID, claims.UserID.String(), issuedAt)
	}

	globalLimiter := httprate.Limit(cfg.RateLimitPlatformCeilingPerMinute, time.Minute, httprate.WithKeyByIP())
	loginLimiter := httprate.Limit(10, time.Minute, httprate.WithKeyByIP())
	if redisClient != nil {
		globalLimiter = httprate.Limit(cfg.RateLimitPlatformCeilingPerMinute, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient, "platform-ceiling")))
		loginLimiter = httprate.Limit(10, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient, "login")))
	}
	tenantLimiter, err := ratelimit.NewTenantLimiter(redisClient, ratelimit.Policy{
		Window:           time.Minute,
		TenantReadLimit:  cfg.RateLimitTenantReadPerMinute,
		ActorReadLimit:   cfg.RateLimitActorReadPerMinute,
		TenantWriteLimit: cfg.RateLimitTenantWritePerMinute,
		ActorWriteLimit:  cfg.RateLimitActorWritePerMinute,
		TenantAgentLimit: cfg.RateLimitTenantAgentPerMinute,
		AgentLimit:       cfg.RateLimitAgentPerMinute,
	})
	if err != nil {
		panic(fmt.Sprintf("tenant rate-limit policy: %v", err))
	}

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
	r.Use(globalLimiter)

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
		assets:                 assets,
		programs:               programs,
		rf:                     rf,
		rfAnomalies:            rfAnomalies,
		satellites:             satellites,
		users:                  users,
		agentKeys:              agentKeys,
		auditLog:               auditLog,
		spaceCVEs:              spaceCVEs,
		assetAlerts:            assetAlerts,
		incidents:              incidents,
		rbac:                   rbac,
		redisClient:            redisClient,
		agentConfigSigner:      agentConfigSigner,
		agentConfigKeyProvider: agentConfigKeyProvider,
		detectionRules:         detectionRules,
		blocklist:              blocklist,
		sessions:               sessions,
		cache:                  cache,
		webAuthnCredentials:    webAuthnCredentials,
		passwordResets:         passwordResets,
		searchStore:            searchStore,
		agentIdentities:        agentIdentities,
		agentConfigs:           agentConfigs,
		fleetRollouts:          fleetRollouts,
		orbitalEvents:          orbitalEvents,
		organizations:          organizations,
		tenantScope:            tenantScope,
		agentCA:                agentCA,
		webAuthnFlows:          webAuthnFlows,
		webAuthnHandler:        webAuthnHandler,
		mailClient:             mailClient,
		isTokenRevoked:         isTokenRevoked,
		loginLimiter:           loginLimiter,
		tenantLimiter:          tenantLimiter,
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authenticatedTenantRateLimit(cfg.JWTSecret, cfg.JWTPreviousSecret, tenantLimiter))
		api.register(r)
	})

	return r
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}

func (w *statusCapturingResponseWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func structuredRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		captured := &statusCapturingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(captured, r)
		status := captured.status
		if status == 0 {
			status = http.StatusOK
		}
		event := map[string]interface{}{
			"event":       "http_request",
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      status,
			"bytes":       captured.bytes,
			"request_id":  middleware.GetReqID(r.Context()),
			"remote_ip":   clientIP(r),
			"duration_ms": time.Since(start).Milliseconds(),
		}
		if encoded, err := json.Marshal(event); err == nil {
			log.Print(string(encoded))
		}
	})
}

func sanitizeRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID != "" && !validRequestID(requestID) {
			r.Header.Del("X-Request-ID")
		}
		next.ServeHTTP(w, r)
	})
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.', ':':
			continue
		default:
			return false
		}
	}
	return true
}

func requestIDResponseHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestID := middleware.GetReqID(r.Context()); requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
		}
		next.ServeHTTP(w, r)
	})
}

func maxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

func pageParams(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return limit, offset
}

func optionalUUID(value, field string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", field)
	}
	return &id, nil
}

func parseIncidentFilter(r *http.Request) (store.IncidentFilter, error) {
	q := r.URL.Query()
	filter := store.IncidentFilter{
		Status:   strings.ToLower(strings.TrimSpace(q.Get("status"))),
		Severity: strings.ToLower(strings.TrimSpace(q.Get("severity"))),
		AgentID:  strings.TrimSpace(q.Get("agent_id")),
	}
	if filter.Status != "" {
		if err := validate.IncidentStatus(filter.Status); err != nil {
			return store.IncidentFilter{}, err
		}
	}
	if filter.Severity != "" {
		if err := validate.IncidentSeverity(filter.Severity); err != nil {
			return store.IncidentFilter{}, err
		}
	}
	if filter.AgentID != "" {
		if err := validate.AgentID(filter.AgentID); err != nil {
			return store.IncidentFilter{}, err
		}
	}
	assignedTo, err := optionalUUID(q.Get("assigned_to"), "assigned_to")
	if err != nil {
		return store.IncidentFilter{}, err
	}
	filter.AssignedTo = assignedTo
	satelliteID, err := optionalUUID(q.Get("satellite_id"), "satellite_id")
	if err != nil {
		return store.IncidentFilter{}, err
	}
	filter.SatelliteID = satelliteID
	return filter, nil
}

func incidentsCacheKey(orgID uuid.UUID, limit, offset int, filter store.IncidentFilter) string {
	assignedTo := ""
	if filter.AssignedTo != nil {
		assignedTo = filter.AssignedTo.String()
	}
	satelliteID := ""
	if filter.SatelliteID != nil {
		satelliteID = filter.SatelliteID.String()
	}
	return fmt.Sprintf("lyrahcle:cache:incidents:%s:%d:%d:%s:%s:%s:%s:%s",
		orgID, limit, offset, filter.Status, filter.Severity, filter.AgentID, assignedTo, satelliteID)
}

func incidentsCacheKeyDefault(orgID uuid.UUID) string {
	return incidentsCacheKey(orgID, 50, 0, store.IncidentFilter{})
}

func assetAlertsCacheKey(orgID uuid.UUID, limit, offset int) string {
	return fmt.Sprintf("lyrahcle:cache:asset-alerts:%s:%d:%d", orgID, limit, offset)
}

func assetAlertsCacheKeyDefault(orgID uuid.UUID) string {
	return assetAlertsCacheKey(orgID, 50, 0)
}

func rfReadingsCacheKey(orgID uuid.UUID) string {
	return fmt.Sprintf("lyrahcle:cache:rf:readings:%s", orgID)
}

func rfAnomaliesCacheKey(orgID uuid.UUID) string {
	return fmt.Sprintf("lyrahcle:cache:rf:anomalies:%s", orgID)
}

func parseSpaceCVEFilter(r *http.Request) store.SpaceCVEFilter {
	q := r.URL.Query()
	filter := store.SpaceCVEFilter{
		Severity: strings.TrimSpace(q.Get("severity")),
		Keyword:  strings.TrimSpace(q.Get("keyword")),
		SortBy:   strings.TrimSpace(q.Get("sort")),
		SortDesc: strings.ToLower(strings.TrimSpace(q.Get("order"))) != "asc",
	}
	if from := strings.TrimSpace(q.Get("published_from")); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.PublishedFrom = t
		}
	}
	if to := strings.TrimSpace(q.Get("published_to")); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filter.PublishedTo = t.Add(24*time.Hour - time.Second)
		}
	}
	return filter
}

func securityHeaders(tlsEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), usb=()")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			if tlsEnabled {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func syncTLEs(ctx context.Context, groups []string, satellites *store.SatelliteStore, policy tlequality.Policy) {
	defer refreshTLEMetrics(ctx, satellites, policy)
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}

		entries, err := celestrak.FetchGroup(group)
		if err != nil {
			metrics.CelestrakSyncFailuresTotal.WithLabelValues(group).Inc()
			log.Printf("tle-sync: fetch group %q: %v", group, err)
			continue
		}

		if len(entries) == 0 {
			metrics.CelestrakSyncFailuresTotal.WithLabelValues(group).Inc()
			log.Printf("tle-sync: group %q returned no entries", group)
			continue
		}

		for _, e := range entries {
			if err := satellites.Upsert(ctx, e.NoradID, e.Name, e.Line1, e.Line2); err != nil {
				metrics.CelestrakSyncFailuresTotal.WithLabelValues(group).Inc()
				log.Printf("tle-sync: upsert %d (%s): %v", e.NoradID, e.Name, err)
			}
		}

		log.Printf("tle-sync: synced %d satellites from group %q", len(entries), group)
	}
}

func refreshTLEMetrics(ctx context.Context, satellites *store.SatelliteStore, policy tlequality.Policy) {
	list, err := satellites.List(ctx)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for _, satellite := range list {
		recordTLEMetrics(satellite.NoradID, tlequality.Assess(satellite.TLEEpoch, now, policy))
	}
}

const anomalyHistoryWindow = 20

func detectRFAnomalies(ctx context.Context, anomalies *store.AnomalyStore, rules *store.DetectionRuleStore, incidents *store.IncidentStore, auditLog *store.AuditStore, assetAlerts *store.AssetAlertStore, satellites *store.SatelliteStore, identities *store.AgentIdentityStore, spaceCVEs *store.SpaceCVEStore, orbitalEvents *store.OrbitalEventStore, policy tlequality.Policy, orbitalAutoEscalation bool, reading store.RFReading, history []store.RFReading) {
	powerHistory := make([]float64, len(history))
	freqHistory := make([]float64, len(history))
	for i, h := range history {
		powerHistory[i] = h.PowerDbm
		freqHistory[i] = float64(h.FrequencyHz)
	}

	checks := []anomaly.Result{
		anomaly.Detect("power_dbm", powerHistory, reading.PowerDbm),
		anomaly.Detect("frequency_hz", freqHistory, float64(reading.FrequencyHz)),
		anomaly.DetectTrend("power_dbm", powerHistory, reading.PowerDbm),
		anomaly.DetectTrend("frequency_hz", freqHistory, float64(reading.FrequencyHz)),
	}

	for _, result := range checks {
		if !result.Anomaly {
			continue
		}
		created, err := anomalies.InsertReturning(ctx, store.RFAnomaly{
			OrganizationID: reading.OrganizationID,
			ReadingID:      reading.ID,
			AgentID:        reading.AgentID,
			SatelliteID:    reading.SatelliteID,
			Metric:         result.Metric,
			Value:          result.Value,
			BaselineMean:   result.Mean,
			BaselineStddev: result.StdDev,
			ZScore:         result.ZScore,
		})
		if err != nil {
			log.Printf("anomaly: failed to record %s anomaly for agent %s: %v", result.Metric, reading.AgentID, err)
			continue
		}
		correlateRFDetectionRules(ctx, rules, incidents, auditLog, assetAlerts, satellites, identities, spaceCVEs, orbitalEvents, policy, orbitalAutoEscalation, reading, created)
	}
}

func correlateRFDetectionRules(ctx context.Context, rules *store.DetectionRuleStore, incidents *store.IncidentStore, auditLog *store.AuditStore, assetAlerts *store.AssetAlertStore, satellites *store.SatelliteStore, identities *store.AgentIdentityStore, spaceCVEs *store.SpaceCVEStore, orbitalEvents *store.OrbitalEventStore, policy tlequality.Policy, orbitalAutoEscalation bool, reading store.RFReading, anomalyRecord store.RFAnomaly) {
	enabledRules, err := rules.ListEnabledByOrganization(ctx, anomalyRecord.OrganizationID)
	if err != nil {
		log.Printf("detection: failed to load rules: %v", err)
		return
	}

	type matchedRule struct {
		rule  detection.Rule
		match store.DetectionMatch
	}
	matched := []matchedRule{}
	autoNames := []string{}
	autoRuleCount := 0
	autoSeverity := ""
	for _, rule := range enabledRules {
		if !detection.Matches(rule, detection.Signal{Metric: anomalyRecord.Metric, ZScore: anomalyRecord.ZScore}) {
			continue
		}
		match, _, err := rules.RecordMatchIfAllowed(ctx, anomalyRecord.OrganizationID, rule, anomalyRecord.ID, anomalyRecord.AgentID, anomalyRecord.Metric, anomalyRecord.ZScore)
		if err != nil {
			log.Printf("detection: failed to record rule %s match: %v", rule.ID, err)
			continue
		}
		if match.ID == uuid.Nil {
			continue
		}
		matched = append(matched, matchedRule{rule: rule, match: match})
		if rule.AutoIncident {
			autoRuleCount++
			if len(autoNames) < 8 {
				autoNames = append(autoNames, rule.Name)
			}
			autoSeverity = detection.HigherSeverity(autoSeverity, rule.Severity)
		}
	}
	if autoRuleCount == 0 {
		return
	}

	orbitalSecurity := buildOrbitalSecurityContext(ctx, satellites, identities, spaceCVEs, policy, orbitalAutoEscalation, reading, anomalyRecord)
	if orbitalSecurity.SeverityFloor != "" {
		autoSeverity = detection.HigherSeverity(autoSeverity, orbitalSecurity.SeverityFloor)
	}

	ruleSummary := strings.Join(autoNames, ", ")
	if autoRuleCount > len(autoNames) {
		ruleSummary = fmt.Sprintf("%s (+%d more)", ruleSummary, autoRuleCount-len(autoNames))
	}
	incident, _, err := incidents.CreateWithResult(ctx, store.CreateIncidentInput{
		OrganizationID:  anomalyRecord.OrganizationID,
		Title:           fmt.Sprintf("RF detection on %s", anomalyRecord.AgentID),
		Description:     fmt.Sprintf("Matched rules: %s. Metric %s produced a z-score of %.2f.", ruleSummary, anomalyRecord.Metric, anomalyRecord.ZScore),
		Severity:        autoSeverity,
		SourceType:      "detection_rule",
		SatelliteID:     anomalyRecord.SatelliteID,
		AgentID:         anomalyRecord.AgentID,
		SourceAnomalyID: &anomalyRecord.ID,
	})
	if err != nil {
		log.Printf("detection: failed to create correlated incident for anomaly %s: %v", anomalyRecord.ID, err)
		return
	}

	matchIDs := make([]uuid.UUID, 0, len(matched))
	for _, item := range matched {
		matchIDs = append(matchIDs, item.match.ID)
		if _, err := incidents.AddEvidence(ctx, incident.ID, store.CreateIncidentEvidenceInput{
			EvidenceType: "detection_rule",
			EvidenceRef:  item.rule.ID.String(),
			Label:        item.rule.Name,
			Metadata: map[string]interface{}{
				"match_id": item.match.ID.String(),
				"metric":   anomalyRecord.Metric,
				"z_score":  anomalyRecord.ZScore,
			},
		}); err != nil {
			log.Printf("detection: failed to attach rule evidence to incident %s: %v", incident.ID, err)
		}
	}
	if _, err := incidents.AddEvidence(ctx, incident.ID, store.CreateIncidentEvidenceInput{
		EvidenceType: "rf_anomaly",
		EvidenceRef:  anomalyRecord.ID.String(),
		Label:        anomalyRecord.Metric,
	}); err != nil {
		log.Printf("detection: failed to attach anomaly evidence to incident %s: %v", incident.ID, err)
	}
	if err := rules.AttachIncidentInOrganization(ctx, anomalyRecord.OrganizationID, matchIDs, incident.ID); err != nil {
		log.Printf("detection: failed to link matches to incident %s: %v", incident.ID, err)
	}
	attachOrbitalSecurityEvidence(ctx, incidents, incident.ID.String(), orbitalSecurity)
	attachCrossDomainCorrelationEvidence(ctx, incidents, auditLog, assetAlerts, orbitalEvents, incident.ID, anomalyRecord.AgentID, anomalyRecord.SatelliteID, len(matched), orbitalSecurity)
	if auditLog != nil {
		details := map[string]interface{}{
			"organization_id": anomalyRecord.OrganizationID.String(),
			"incident_id":     incident.ID.String(),
			"reference":       incident.Reference,
			"anomaly_id":      anomalyRecord.ID.String(),
			"rule_count":      len(matched),
		}
		dedupeKey := "correlation-incident:" + incident.ID.String()
		if _, err := auditLog.InsertIdempotentInOrganization(ctx, anomalyRecord.OrganizationID, dedupeKey, audit.ActorSystem, "correlation-engine", audit.ActionIncidentCreated, "", details); err != nil {
			log.Printf("detection: failed to record idempotent incident audit for %s: %v", incident.ID, err)
		}
	}
}

func validateDetectionRule(name, description, metric, operator string, threshold float64, severity string, cooldownSeconds int) error {
	if err := validate.DetectionRuleName(name); err != nil {
		return err
	}
	if err := validate.DetectionRuleDescription(description); err != nil {
		return err
	}
	if err := validate.DetectionRuleMetric(metric); err != nil {
		return err
	}
	if err := validate.DetectionRuleOperator(operator); err != nil {
		return err
	}
	if err := validate.DetectionRuleThreshold(threshold); err != nil {
		return err
	}
	if err := validate.IncidentSeverity(severity); err != nil {
		return err
	}
	return validate.DetectionRuleCooldown(cooldownSeconds)
}

func validateAgentDeviceReport(info store.AgentDeviceInfo) error {
	if !info.ReportedAt.IsZero() || info.Compatibility != "" || info.CompatibilityReason != "" {
		return errors.New("reported_at and compatibility fields are read-only")
	}
	for name, value := range map[string]string{
		"driver": info.Driver, "hardware": info.Hardware, "manufacturer": info.Manufacturer,
		"product": info.Product, "serial": info.Serial,
	} {
		if len(strings.TrimSpace(value)) > 256 {
			return fmt.Errorf("%s is too long", name)
		}
	}
	c := info.Capabilities
	if c.IntegrityBaselineSHA256 != "" || c.IntegrityStatus != "" {
		return errors.New("integrity baseline and status are read-only")
	}
	for _, digest := range []string{c.IntegrityManifestSHA256, c.IntegrityMachineIDHash} {
		if digest != "" {
			if len(digest) != 64 {
				return errors.New("invalid integrity SHA-256 digest")
			}
			if _, err := hex.DecodeString(digest); err != nil {
				return errors.New("invalid integrity SHA-256 digest")
			}
		}
	}
	if len(c.IntegrityBootID) > 64 || c.IntegrityFileCount < 0 || c.IntegrityFileCount > 1024 {
		return errors.New("invalid edge integrity metadata")
	}
	if len(c.Architecture) > 64 || len(c.OS) > 64 || len(c.DriverVersion) > 128 || len(c.FirmwareVersion) > 128 {
		return errors.New("hardware capability metadata is too long")
	}
	if c.FrequencyMinHz < 0 || c.FrequencyMaxHz < 0 || (c.FrequencyMaxHz > 0 && c.FrequencyMinHz > c.FrequencyMaxHz) {
		return errors.New("invalid frequency capability range")
	}
	if c.Channels < 0 || c.Channels > 64 || c.SampleDrops < 0 || len(c.SampleRatesHz) > 64 || len(c.BandwidthsHz) > 64 || len(c.GainElements) > 64 {
		return errors.New("invalid hardware capabilities")
	}
	for _, value := range append(append([]int64{}, c.SampleRatesHz...), c.BandwidthsHz...) {
		if value <= 0 {
			return errors.New("sample rates and bandwidths must be positive")
		}
	}
	for _, name := range c.GainElements {
		if len(strings.TrimSpace(name)) > 64 {
			return errors.New("gain element name is too long")
		}
	}
	if c.TemperatureC != nil && (*c.TemperatureC < -100 || *c.TemperatureC > 250) {
		return errors.New("invalid hardware temperature")
	}
	return nil
}

func logAuditInOrganization(ctx context.Context, auditLog *store.AuditStore, organizationID uuid.UUID, actorType, actorID, action, ip string, details map[string]interface{}) {
	if organizationID == uuid.Nil {
		organizationID = store.DefaultOrganizationID
	}
	if err := auditLog.InsertInOrganization(ctx, organizationID, actorType, actorID, action, ip, details); err != nil {
		log.Printf("audit: failed to record %s by %s %s in organization %s: %v", action, actorType, actorID, organizationID, err)
	}
}

func logAudit(ctx context.Context, auditLog *store.AuditStore, actorType, actorID, action, ip string, details map[string]interface{}) {
	organizationID := store.DefaultOrganizationID
	if claims, ok := auth.FromContext(ctx); ok && claims.OrganizationID != uuid.Nil {
		organizationID = claims.OrganizationID
	} else if details != nil {
		switch value := details["organization_id"].(type) {
		case uuid.UUID:
			if value != uuid.Nil {
				organizationID = value
			}
		case string:
			if parsed, err := uuid.Parse(value); err == nil && parsed != uuid.Nil {
				organizationID = parsed
			}
		}
	}
	if err := auditLog.InsertInOrganization(ctx, organizationID, actorType, actorID, action, ip, details); err != nil {
		log.Printf("audit: failed to record %s by %s %s: %v", action, actorType, actorID, err)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type satWithPos struct {
	store.Satellite
	Position          *orbit.Position `json:"position,omitempty"`
	TLEStatus         string          `json:"tle_status"`
	TLEAgeSeconds     int64           `json:"tle_age_seconds"`
	OrbitalConfidence string          `json:"orbital_confidence"`
}

func satellitesWithPositions(ctx context.Context, satellites *store.SatelliteStore, cache *respcache.Cache, policy tlequality.Policy) ([]satWithPos, error) {
	var out []satWithPos
	if hit, _ := cache.Get(ctx, satellitesCacheKey, &out); hit {
		return out, nil
	}

	list, err := satellites.List(ctx)
	if err != nil {
		return nil, err
	}

	out = make([]satWithPos, len(list))
	workers := 8
	if len(list) < workers {
		workers = len(list)
	}
	if workers > 0 {
		jobs := make(chan int)
		var wg sync.WaitGroup
		for worker := 0; worker < workers; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for index := range jobs {
					sat := list[index]
					assessment := tlequality.Assess(sat.TLEEpoch, time.Now().UTC(), policy)
					recordTLEMetrics(sat.NoradID, assessment)
					entry := satWithPos{
						Satellite:         sat,
						TLEStatus:         string(assessment.Status),
						TLEAgeSeconds:     int64(assessment.Age.Seconds()),
						OrbitalConfidence: assessment.Confidence,
					}
					if assessment.Status == tlequality.StatusFresh || assessment.Status == tlequality.StatusAging {
						if pos, err := orbit.CurrentPosition(sat.TLELine1, sat.TLELine2); err == nil {
							entry.Position = &pos
						}
					}
					out[index] = entry
				}
			}()
		}
		for index := range list {
			jobs <- index
		}
		close(jobs)
		wg.Wait()
	}

	cache.Set(ctx, satellitesCacheKey, out, 5*time.Second)
	return out, nil
}

func tlePolicy(cfg config.Config) tlequality.Policy {
	return tlequality.Policy{CorrelationMaxAge: cfg.TLECorrelationMaxAge, StaleMaxAge: cfg.TLEStaleMaxAge}
}

func recordTLEMetrics(noradID int, assessment tlequality.Assessment) {
	label := strconv.Itoa(noradID)
	metrics.TLEAgeSeconds.WithLabelValues(label).Set(assessment.Age.Seconds())
	stale := 0.0
	if assessment.Status == tlequality.StatusStale || assessment.Status == tlequality.StatusUnusable {
		stale = 1
	}
	metrics.TLEStale.WithLabelValues(label).Set(stale)
	eligible := 0.0
	if assessment.EscalationAllowed {
		eligible = 1
	}
	metrics.TLECorrelationEligible.WithLabelValues(label).Set(eligible)
}

func seedAdmin(ctx context.Context, users *store.UserStore, cfg config.Config) error {
	count, err := users.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := auth.HashPassword(cfg.SeedAdminPass)
	if err != nil {
		return err
	}

	seeded, err := users.CreateProvisioned(ctx, cfg.SeedAdminEmail, hash, string(auth.RoleAdmin))
	if err != nil {
		return err
	}
	if err := users.GrantPlatformAdministrator(ctx, seeded.ID); err != nil {
		return err
	}

	log.Println("============================================================")
	log.Printf("Created default admin account: %q / (seed password)", cfg.SeedAdminEmail)
	log.Println("This is a well-known default and MUST be changed immediately:")
	log.Println("  POST /api/v1/auth/change-password (after logging in)")
	log.Println("Override the seed via SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD")
	log.Println("before first boot in any non-local environment.")
	log.Println("============================================================")

	return nil
}
