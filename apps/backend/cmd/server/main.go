package main

import (
	"context"
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

	"github.com/mlko01/lyrahcle/apps/backend/internal/alerts"
	"github.com/mlko01/lyrahcle/apps/backend/internal/anomaly"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/celestrak"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/db"
	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
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
	"github.com/mlko01/lyrahcle/apps/backend/internal/tmtc"
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

	sched := scheduler.New()
	_ = sched.Every("0 */6 * * *", "recon-sweep", func() {
		log.Println("recon sweep tick (not yet implemented)")
	})
	_ = sched.Every("0 */6 * * *", "tle-sync", func() {
		syncTLEs(ctx, cfg.CelestrakGroups, satellites, tlePolicy(cfg))
	})
	sched.Start()
	defer sched.Stop()

	go syncTLEs(ctx, cfg.CelestrakGroups, satellites, tlePolicy(cfg))

	r := newRouter(cfg, pool, assets, programs, rf, rfAnomalies, satellites, users, agentKeys, auditLog, spaceCVEs, assetAlerts, incidents, redisClient)

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
			_ = srv.Close()
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
	redisClient *redis.Client,
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
	agentCA, err := buildAgentCA(cfg)
	if err != nil {
		panic(fmt.Sprintf("agent CA configuration: %v", err))
	}
	agentConfigSigner, err := buildAgentConfigSigner(cfg)
	if err != nil {
		panic(fmt.Sprintf("agent config signing configuration: %v", err))
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
		if user.MustChangePassword != claims.PasswordChangeRequired {
			return true, nil
		}
		issuedAt := time.Unix(0, claims.SessionIssuedAt).UTC()
		return blocklist.Check(ctx, claims.ID, claims.UserID.String(), issuedAt)
	}

	globalLimiter := httprate.Limit(120, time.Minute, httprate.WithKeyByIP())
	loginLimiter := httprate.Limit(10, time.Minute, httprate.WithKeyByIP())
	if redisClient != nil {
		globalLimiter = httprate.Limit(120, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient, "global")))
		loginLimiter = httprate.Limit(10, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient, "login")))
	}

	r := chi.NewRouter()
	r.Use(sanitizeRequestID)
	r.Use(middleware.RequestID)
	r.Use(requestIDResponseHeader)
	r.Use(structuredRequestLogger)
	r.Use(observability.Middleware)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders(cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""))
	r.Use(metrics.Middleware)
	r.Use(maxBodySize(1 << 20))
	r.Use(globalLimiter)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"lyrahcle-backend"}`))
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
			w.Write([]byte(`{"status":"not_ready"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/docs", serveAPIDocs)

	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"message":"pong from lyrahcle"}`))
		})

		registerAgentIdentityRoutes(r, cfg, agentCA, agentIdentities, agentKeys, auditLog, isTokenRevoked)
		registerAgentConfigRoutes(r, cfg, agentConfigSigner, agentConfigs, agentIdentities, satellites, auditLog, isTokenRevoked)

		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/search", func(w http.ResponseWriter, r *http.Request) {
			query := strings.TrimSpace(r.URL.Query().Get("q"))
			if len(query) < 2 || len(query) > 96 {
				http.Error(w, "q must contain 2-96 characters", http.StatusBadRequest)
				return
			}
			results, err := searchStore.Search(r.Context(), query, 12)
			if err != nil {
				http.Error(w, "search failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		})

		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/system-info", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"environment":             cfg.Environment,
				"license":                 "AGPLv3 (core)",
				"celestrak_groups":        cfg.CelestrakGroups,
				"celestrak_sync_schedule": "every 6 hours",
				"nvd_sync_note":           "managed by the cve-sync worker (apps/cve-sync)",
			})
		})

		r.Route("/assets", func(r chi.Router) {
			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				limit, offset := pageParams(r)
				page, err := assets.List(r.Context(), limit, offset)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleOperator, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					ProgramID string `json:"program_id"`
					Kind      string `json:"kind"`
					Value     string `json:"value"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Value = strings.TrimSpace(body.Value)

				programID, err := uuid.Parse(body.ProgramID)
				if err != nil {
					http.Error(w, "program_id must be a valid UUID", http.StatusBadRequest)
					return
				}
				if err := validate.AssetKind(body.Kind); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.AssetValue(body.Value); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				if err := assets.Upsert(r.Context(), programID, body.Kind, body.Value); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)
			})
		})

		r.Route("/programs", func(r chi.Router) {
			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := programs.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleOperator, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Name string `json:"name"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Name = strings.TrimSpace(body.Name)
				if err := validate.ProgramName(body.Name); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				program, err := programs.Create(r.Context(), body.Name)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(program)
			})
		})

		r.Route("/auth", func(r chi.Router) {
			r.With(loginLimiter).Post("/login", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Identifier string `json:"identifier"`
					Password   string `json:"password"`
					TOTPCode   string `json:"totp_code"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Identifier = strings.TrimSpace(strings.ToLower(body.Identifier))
				if body.Identifier == "" || body.Password == "" {
					http.Error(w, "invalid credentials", http.StatusUnauthorized)
					return
				}

				user, err := users.GetByIdentifier(r.Context(), body.Identifier)
				if err != nil || !auth.CheckPassword(user.PasswordHash, body.Password) || !user.Enabled {
					metrics.AuthLoginsTotal.WithLabelValues("invalid_credentials").Inc()
					logAudit(r.Context(), auditLog, audit.ActorAnonymous, body.Identifier, audit.ActionLoginFailed, clientIP(r), nil)
					http.Error(w, "invalid credentials", http.StatusUnauthorized)
					return
				}

				webAuthnCount, err := webAuthnCredentialCount(r.Context(), webAuthnCredentials, user.ID)
				if err != nil {
					http.Error(w, "authentication state unavailable", http.StatusServiceUnavailable)
					return
				}
				if webAuthnCount > 0 {
					metrics.AuthLoginsTotal.WithLabelValues("webauthn_required").Inc()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "webauthn_required"})
					return
				}

				if user.TOTPEnabled {
					if body.TOTPCode == "" {
						metrics.AuthLoginsTotal.WithLabelValues("totp_required").Inc()
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						json.NewEncoder(w).Encode(map[string]string{"error": "totp_required"})
						return
					}
					if !auth.VerifyTOTP(user.TOTPSecret, body.TOTPCode) {
						metrics.AuthLoginsTotal.WithLabelValues("invalid_totp").Inc()
						logAudit(r.Context(), auditLog, audit.ActorAnonymous, body.Identifier, audit.ActionLoginFailed, clientIP(r), map[string]interface{}{"reason": "invalid_totp"})
						http.Error(w, "invalid two-factor code", http.StatusUnauthorized)
						return
					}
				}

				token, tokenClaims, err := auth.IssueUserTokenWithState(cfg.JWTSecret, user.ID, auth.Role(user.Role), user.MustChangePassword, user.AuthVersion, cfg.JWTAccessTTL)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if redisClient != nil {
					if err := sessions.Register(r.Context(), sessionstore.Session{
						ID:        tokenClaims.ID,
						UserID:    user.ID.String(),
						CreatedAt: tokenClaims.IssuedAt.Time.UTC(),
						ExpiresAt: tokenClaims.ExpiresAt.Time.UTC(),
						IP:        clientIP(r),
						UserAgent: r.UserAgent(),
					}); err != nil {
						http.Error(w, "session registration failed", http.StatusServiceUnavailable)
						return
					}
				}

				metrics.AuthLoginsTotal.WithLabelValues("success").Inc()
				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionLogin, clientIP(r), map[string]interface{}{"email": user.Email})

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"token": token, "role": user.Role, "must_change_password": user.MustChangePassword})
			})

			registerWebAuthnRoutes(r, cfg, webAuthnHandler, webAuthnFlows, webAuthnCredentials, users, sessions, blocklist, auditLog, redisClient, isTokenRevoked, loginLimiter)
			registerPasswordRecoveryRoutes(r, cfg, users, passwordResets, webAuthnCredentials, sessions, blocklist, auditLog, mailClient, redisClient, loginLimiter)

			r.Route("/totp", func(r chi.Router) {
				r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))

				r.Post("/setup", func(w http.ResponseWriter, r *http.Request) {
					claims, ok := auth.FromContext(r.Context())
					if !ok {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					user, err := users.GetByID(r.Context(), claims.UserID)
					if err != nil {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if user.TOTPEnabled {
						http.Error(w, "two-factor authentication is already enabled", http.StatusConflict)
						return
					}

					secret, err := auth.GenerateTOTPSecret()
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if err := users.SetPendingTOTPSecret(r.Context(), user.ID, secret); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					uri := auth.TOTPProvisioningURI("Lyrahcle", user.Email, secret)
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{"provider": "google_authenticator", "issuer": "Lyrahcle", "account_name": user.Email, "secret": secret, "otpauth_url": uri})
				})

				r.Post("/enable", func(w http.ResponseWriter, r *http.Request) {
					var body struct {
						Code string `json:"code"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						http.Error(w, "invalid JSON body", http.StatusBadRequest)
						return
					}
					claims, ok := auth.FromContext(r.Context())
					if !ok {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					user, err := users.GetByID(r.Context(), claims.UserID)
					if err != nil {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if user.TOTPSecret == "" {
						http.Error(w, "call /auth/totp/setup first", http.StatusBadRequest)
						return
					}
					if !auth.VerifyTOTP(user.TOTPSecret, body.Code) {
						http.Error(w, "invalid two-factor code", http.StatusUnauthorized)
						return
					}
					if err := blocklist.RevokeUser(r.Context(), user.ID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
						http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
						return
					}
					if redisClient != nil {
						if err := sessions.DeleteUser(r.Context(), user.ID.String()); err != nil {
							http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
							return
						}
					}
					if err := users.EnableTOTP(r.Context(), user.ID); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionTOTPEnabled, clientIP(r), nil)
					w.WriteHeader(http.StatusNoContent)
				})

				r.Post("/disable", func(w http.ResponseWriter, r *http.Request) {
					var body struct {
						Password string `json:"password"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						http.Error(w, "invalid JSON body", http.StatusBadRequest)
						return
					}
					claims, ok := auth.FromContext(r.Context())
					if !ok {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					user, err := users.GetByID(r.Context(), claims.UserID)
					if err != nil {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if !auth.CheckPassword(user.PasswordHash, body.Password) {
						http.Error(w, "invalid password", http.StatusUnauthorized)
						return
					}
					if err := blocklist.RevokeUser(r.Context(), user.ID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
						http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
						return
					}
					if redisClient != nil {
						if err := sessions.DeleteUser(r.Context(), user.ID.String()); err != nil {
							http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
							return
						}
					}
					if err := users.DisableTOTP(r.Context(), user.ID); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionTOTPDisabled, clientIP(r), nil)
					w.WriteHeader(http.StatusNoContent)
				})
			})

			r.Route("/sessions", func(r chi.Router) {
				r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))

				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					claims, ok := auth.FromContext(r.Context())
					if !ok {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if redisClient == nil {
						http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
						return
					}
					list, err := sessions.List(r.Context(), claims.UserID.String())
					if err != nil {
						http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
						return
					}
					type sessionResponse struct {
						ID        string    `json:"id"`
						CreatedAt time.Time `json:"created_at"`
						ExpiresAt time.Time `json:"expires_at"`
						IP        string    `json:"ip"`
						UserAgent string    `json:"user_agent"`
						Current   bool      `json:"current"`
					}
					out := make([]sessionResponse, 0, len(list))
					for _, item := range list {
						out = append(out, sessionResponse{ID: item.ID, CreatedAt: item.CreatedAt, ExpiresAt: item.ExpiresAt, IP: item.IP, UserAgent: item.UserAgent, Current: item.ID == claims.ID})
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(out)
				})

				r.Delete("/{sessionID}", func(w http.ResponseWriter, r *http.Request) {
					claims, ok := auth.FromContext(r.Context())
					if !ok {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if redisClient == nil {
						http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
						return
					}
					sessionID := strings.TrimSpace(chi.URLParam(r, "sessionID"))
					if _, err := uuid.Parse(sessionID); err != nil {
						http.Error(w, "sessionID must be a valid UUID", http.StatusBadRequest)
						return
					}
					item, err := sessions.Get(r.Context(), sessionID)
					if err != nil || item.UserID != claims.UserID.String() {
						http.Error(w, "session not found", http.StatusNotFound)
						return
					}
					remaining := time.Until(item.ExpiresAt)
					if remaining > 0 {
						if err := blocklist.Revoke(r.Context(), item.ID, remaining); err != nil {
							http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
							return
						}
					}
					if err := sessions.Delete(r.Context(), claims.UserID.String(), item.ID); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionSessionRevoked, clientIP(r), map[string]interface{}{"session_id": item.ID, "current": item.ID == claims.ID})
					w.WriteHeader(http.StatusNoContent)
				})

				r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
					claims, ok := auth.FromContext(r.Context())
					if !ok {
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if redisClient == nil {
						http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
						return
					}
					if err := blocklist.RevokeUser(r.Context(), claims.UserID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
					if err := sessions.DeleteUser(r.Context(), claims.UserID.String()); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionSessionsRevoked, clientIP(r), nil)
					w.WriteHeader(http.StatusNoContent)
				})
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				user, err := users.GetByID(r.Context(), claims.UserID)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":                   user.ID.String(),
					"email":                user.Email,
					"username":             user.Username,
					"role":                 user.Role,
					"totp_enabled":         user.TOTPEnabled,
					"must_change_password": user.MustChangePassword,
					"enabled":              user.Enabled,
				})
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Post("/logout", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				remaining := time.Until(claims.ExpiresAt.Time)
				if err := blocklist.Revoke(r.Context(), claims.ID, remaining); err != nil {
					http.Error(w, "logout could not revoke the session", http.StatusServiceUnavailable)
					return
				}
				if redisClient != nil {
					if err := sessions.Delete(r.Context(), claims.UserID.String(), claims.ID); err != nil {
						http.Error(w, "logout could not remove the session", http.StatusServiceUnavailable)
						return
					}
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionLogout, clientIP(r), nil)
				w.WriteHeader(http.StatusNoContent)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Post("/change-password", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					CurrentPassword string `json:"current_password"`
					NewPassword     string `json:"new_password"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}

				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				user, err := users.GetByID(r.Context(), claims.UserID)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if !auth.CheckPassword(user.PasswordHash, body.CurrentPassword) {
					http.Error(w, "current password is incorrect", http.StatusUnauthorized)
					return
				}
				if err := validate.Password(body.NewPassword); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if auth.CheckPassword(user.PasswordHash, body.NewPassword) {
					http.Error(w, "new password must differ from the current password", http.StatusBadRequest)
					return
				}

				newHash, err := auth.HashPassword(body.NewPassword)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := blocklist.RevokeUser(r.Context(), user.ID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
					http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
					return
				}
				if redisClient != nil {
					if err := sessions.DeleteUser(r.Context(), user.ID.String()); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
				}
				if err := users.UpdatePassword(r.Context(), user.ID, newHash); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionPasswordChanged, clientIP(r), map[string]interface{}{"email": user.Email})

				w.WriteHeader(http.StatusNoContent)
			})
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin, isTokenRevoked))

			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := users.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Email    string `json:"email"`
					Password string `json:"password"`
					Role     string `json:"role"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Email = strings.TrimSpace(strings.ToLower(body.Email))
				if err := validate.Email(body.Email); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.Password(body.Password); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if body.Role == "" {
					body.Role = string(auth.RoleViewer)
				}
				if !auth.ValidRole(body.Role) {
					http.Error(w, "role must be one of: viewer, auditor, analyst, operator, admin", http.StatusBadRequest)
					return
				}

				hash, err := auth.HashPassword(body.Password)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				user, err := users.Create(r.Context(), body.Email, hash, body.Role)
				if err != nil {
					if err == store.ErrEmailTaken {
						http.Error(w, "email already registered", http.StatusConflict)
						return
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if claims, ok := auth.FromContext(r.Context()); ok {
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionUserCreated, clientIP(r), map[string]interface{}{"target_email": user.Email, "role": user.Role})
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{"id": user.ID.String(), "email": user.Email, "username": user.Username, "role": user.Role, "enabled": user.Enabled, "must_change_password": user.MustChangePassword})
			})

			r.Patch("/{userID}/role", func(w http.ResponseWriter, r *http.Request) {
				userID, err := uuid.Parse(chi.URLParam(r, "userID"))
				if err != nil {
					http.Error(w, "userID must be a valid UUID", http.StatusBadRequest)
					return
				}

				var body struct {
					Role string `json:"role"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if !auth.ValidRole(body.Role) {
					http.Error(w, "role must be one of: viewer, auditor, analyst, operator, admin", http.StatusBadRequest)
					return
				}

				target, err := users.GetByID(r.Context(), userID)
				if err != nil {
					if err == store.ErrUserNotFound {
						http.Error(w, "user not found", http.StatusNotFound)
						return
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if target.Role == string(auth.RoleAdmin) && target.Enabled && body.Role != string(auth.RoleAdmin) {
					adminCount, err := users.CountAdmins(r.Context())
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if adminCount <= 1 {
						http.Error(w, "cannot demote the last remaining admin", http.StatusConflict)
						return
					}
				}

				if err := blocklist.RevokeUser(r.Context(), userID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
					http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
					return
				}
				if redisClient != nil {
					if err := sessions.DeleteUser(r.Context(), userID.String()); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
				}

				updated, err := users.UpdateRole(r.Context(), userID, body.Role)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if claims, ok := auth.FromContext(r.Context()); ok {
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionRoleChanged, clientIP(r), map[string]interface{}{"target_user_id": updated.ID.String(), "target_email": updated.Email, "new_role": updated.Role})
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"id": updated.ID.String(), "email": updated.Email, "role": updated.Role, "enabled": updated.Enabled})
			})

			r.Patch("/{userID}/status", func(w http.ResponseWriter, r *http.Request) {
				userID, err := uuid.Parse(chi.URLParam(r, "userID"))
				if err != nil {
					http.Error(w, "userID must be a valid UUID", http.StatusBadRequest)
					return
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				var body struct {
					Enabled bool `json:"enabled"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				target, err := users.GetByID(r.Context(), userID)
				if err != nil {
					http.Error(w, "user not found", http.StatusNotFound)
					return
				}
				if claims.UserID == userID && !body.Enabled {
					http.Error(w, "cannot disable the current user", http.StatusConflict)
					return
				}
				if target.Role == string(auth.RoleAdmin) && target.Enabled && !body.Enabled {
					count, err := users.CountAdmins(r.Context())
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if count <= 1 {
						http.Error(w, "cannot disable the last remaining admin", http.StatusConflict)
						return
					}
				}
				if !body.Enabled {
					if err := blocklist.RevokeUser(r.Context(), userID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
						http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
						return
					}
					if redisClient != nil {
						if err := sessions.DeleteUser(r.Context(), userID.String()); err != nil {
							http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
							return
						}
					}
				}
				updated, err := users.SetEnabled(r.Context(), userID, body.Enabled)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionUserStatusChanged, clientIP(r), map[string]interface{}{"target_user_id": userID.String(), "enabled": body.Enabled})
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(updated)
			})

			r.Delete("/{userID}", func(w http.ResponseWriter, r *http.Request) {
				userID, err := uuid.Parse(chi.URLParam(r, "userID"))
				if err != nil {
					http.Error(w, "userID must be a valid UUID", http.StatusBadRequest)
					return
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if claims.UserID == userID {
					http.Error(w, "cannot delete the current user", http.StatusConflict)
					return
				}
				target, err := users.GetByID(r.Context(), userID)
				if err != nil {
					http.Error(w, "user not found", http.StatusNotFound)
					return
				}
				if target.Role == string(auth.RoleAdmin) && target.Enabled {
					count, err := users.CountAdmins(r.Context())
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if count <= 1 {
						http.Error(w, "cannot delete the last remaining admin", http.StatusConflict)
						return
					}
				}
				if err := blocklist.RevokeUser(r.Context(), userID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
					http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
					return
				}
				if redisClient != nil {
					if err := sessions.DeleteUser(r.Context(), userID.String()); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
				}
				if err := users.SoftDelete(r.Context(), userID); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionUserDeleted, clientIP(r), map[string]interface{}{"target_user_id": userID.String(), "target_email": target.Email})
				w.WriteHeader(http.StatusNoContent)
			})
		})

		r.Route("/agent-keys", func(r chi.Router) {
			r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin, isTokenRevoked))

			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := agentKeys.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					AgentID string `json:"agent_id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.AgentID = strings.TrimSpace(body.AgentID)
				if err := validate.AgentID(body.AgentID); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				plaintext, hash, err := auth.GenerateAgentKey()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := agentKeys.Create(r.Context(), body.AgentID, hash); err != nil {
					if err == store.ErrAgentIDTaken {
						http.Error(w, "agent_id already has an active key; revoke it first", http.StatusConflict)
						return
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if claims, ok := auth.FromContext(r.Context()); ok {
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentKeyCreated, clientIP(r), map[string]interface{}{"agent_id": body.AgentID})
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]string{
					"agent_id": body.AgentID,
					"api_key":  plaintext,
				})
			})

			r.Delete("/{agentID}", func(w http.ResponseWriter, r *http.Request) {
				agentID := chi.URLParam(r, "agentID")
				if _, err := agentKeys.Revoke(r.Context(), agentID); err != nil {
					if err == store.ErrAgentKeyNotFound {
						http.Error(w, "no active key for that agent_id", http.StatusNotFound)
						return
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if claims, ok := auth.FromContext(r.Context()); ok {
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentKeyRevoked, clientIP(r), map[string]interface{}{"agent_id": agentID})
				}

				w.WriteHeader(http.StatusNoContent)
			})
		})

		auditedAgentLookup := func(ctx context.Context, keyHash string) (string, error) {
			agentID, err := agentKeys.AgentIDForHash(ctx, keyHash)
			if err != nil {
				logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentKeyAuthFailed, "", nil)
			}
			return agentID, err
		}
		auditedCertificateLookup := func(ctx context.Context, fingerprint string) (string, error) {
			agentID, err := agentIdentities.AgentIDForCertificate(ctx, fingerprint)
			if err != nil {
				logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentCertificateAuthFailed, "", nil)
			}
			return agentID, err
		}
		meAgentAuth := auth.RequireAgentCredentialOrRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleOperator, auditedAgentLookup, auditedCertificateLookup, isTokenRevoked)

		r.With(meAgentAuth).Post("/agents/me/device", func(w http.ResponseWriter, r *http.Request) {
			agentID, ok := auth.AgentIDFromContext(r.Context())
			if !ok {
				http.Error(w, "this endpoint requires agent key authentication", http.StatusUnauthorized)
				return
			}

			var body store.AgentDeviceInfo
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}

			if err := agentIdentities.ReportDevice(r.Context(), agentID, body); err != nil {
				if errors.Is(err, store.ErrAgentIdentityNotFound) {
					if err := agentKeys.ReportDevice(r.Context(), agentID, body); err != nil {
						http.Error(w, "unknown or inactive agent", http.StatusUnauthorized)
						return
					}
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			w.WriteHeader(http.StatusNoContent)
		})

		r.Route("/detection-rules", func(r chi.Router) {
			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				limit, offset := pageParams(r)
				page, err := detectionRules.List(r.Context(), limit, offset)
				if err != nil {
					http.Error(w, "failed to fetch detection rules", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Name            string  `json:"name"`
					Description     string  `json:"description"`
					Enabled         *bool   `json:"enabled"`
					Metric          string  `json:"metric"`
					Operator        string  `json:"operator"`
					Threshold       float64 `json:"threshold"`
					Severity        string  `json:"severity"`
					CooldownSeconds int     `json:"cooldown_seconds"`
					AutoIncident    *bool   `json:"auto_incident"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Name = strings.TrimSpace(body.Name)
				body.Description = strings.TrimSpace(body.Description)
				body.Metric = strings.ToLower(strings.TrimSpace(body.Metric))
				body.Operator = strings.ToLower(strings.TrimSpace(body.Operator))
				body.Severity = strings.ToLower(strings.TrimSpace(body.Severity))
				if err := validateDetectionRule(body.Name, body.Description, body.Metric, body.Operator, body.Threshold, body.Severity, body.CooldownSeconds); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				enabled := true
				if body.Enabled != nil {
					enabled = *body.Enabled
				}
				if enabled {
					count, err := detectionRules.EnabledCount(r.Context())
					if err != nil {
						http.Error(w, "failed to validate detection rule capacity", http.StatusInternalServerError)
						return
					}
					if count >= store.MaxEnabledDetectionRules {
						http.Error(w, "maximum enabled detection rules reached", http.StatusConflict)
						return
					}
				}
				autoIncident := true
				if body.AutoIncident != nil {
					autoIncident = *body.AutoIncident
				}
				rule, err := detectionRules.Create(r.Context(), detection.Rule{
					Name: body.Name, Description: body.Description, Enabled: enabled, Metric: body.Metric,
					Operator: body.Operator, Threshold: body.Threshold, Severity: body.Severity,
					CooldownSeconds: body.CooldownSeconds, AutoIncident: autoIncident,
				}, claims.UserID)
				if err != nil {
					http.Error(w, "failed to create detection rule", http.StatusConflict)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionDetectionRuleCreated, clientIP(r), map[string]interface{}{"rule_id": rule.ID.String(), "name": rule.Name})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(rule)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin, isTokenRevoked)).Patch("/{ruleID}", func(w http.ResponseWriter, r *http.Request) {
				ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
				if err != nil {
					http.Error(w, "ruleID must be a valid UUID", http.StatusBadRequest)
					return
				}
				rule, err := detectionRules.Get(r.Context(), ruleID)
				if err != nil {
					if err == store.ErrDetectionRuleNotFound {
						http.Error(w, "detection rule not found", http.StatusNotFound)
						return
					}
					http.Error(w, "failed to fetch detection rule", http.StatusInternalServerError)
					return
				}
				wasEnabled := rule.Enabled
				var body struct {
					Name            *string  `json:"name"`
					Description     *string  `json:"description"`
					Enabled         *bool    `json:"enabled"`
					Metric          *string  `json:"metric"`
					Operator        *string  `json:"operator"`
					Threshold       *float64 `json:"threshold"`
					Severity        *string  `json:"severity"`
					CooldownSeconds *int     `json:"cooldown_seconds"`
					AutoIncident    *bool    `json:"auto_incident"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if body.Name != nil {
					rule.Name = strings.TrimSpace(*body.Name)
				}
				if body.Description != nil {
					rule.Description = strings.TrimSpace(*body.Description)
				}
				if body.Enabled != nil {
					rule.Enabled = *body.Enabled
				}
				if body.Metric != nil {
					rule.Metric = strings.ToLower(strings.TrimSpace(*body.Metric))
				}
				if body.Operator != nil {
					rule.Operator = strings.ToLower(strings.TrimSpace(*body.Operator))
				}
				if body.Threshold != nil {
					rule.Threshold = *body.Threshold
				}
				if body.Severity != nil {
					rule.Severity = strings.ToLower(strings.TrimSpace(*body.Severity))
				}
				if body.CooldownSeconds != nil {
					rule.CooldownSeconds = *body.CooldownSeconds
				}
				if body.AutoIncident != nil {
					rule.AutoIncident = *body.AutoIncident
				}
				if err := validateDetectionRule(rule.Name, rule.Description, rule.Metric, rule.Operator, rule.Threshold, rule.Severity, rule.CooldownSeconds); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if rule.Enabled && !wasEnabled {
					count, err := detectionRules.EnabledCount(r.Context())
					if err != nil {
						http.Error(w, "failed to validate detection rule capacity", http.StatusInternalServerError)
						return
					}
					if count >= store.MaxEnabledDetectionRules {
						http.Error(w, "maximum enabled detection rules reached", http.StatusConflict)
						return
					}
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				rule, err = detectionRules.Update(r.Context(), rule, claims.UserID)
				if err != nil {
					http.Error(w, "failed to update detection rule", http.StatusInternalServerError)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionDetectionRuleUpdated, clientIP(r), map[string]interface{}{"rule_id": rule.ID.String(), "name": rule.Name, "enabled": rule.Enabled})
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(rule)
			})
		})

		r.Route("/rf", func(r chi.Router) {
			auditedLookup := func(ctx context.Context, keyHash string) (string, error) {
				agentID, err := agentKeys.AgentIDForHash(ctx, keyHash)
				if err != nil {
					logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentKeyAuthFailed, "", nil)
				}
				return agentID, err
			}
			auditedCertificateLookup := func(ctx context.Context, fingerprint string) (string, error) {
				agentID, err := agentIdentities.AgentIDForCertificate(ctx, fingerprint)
				if err != nil {
					logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentCertificateAuthFailed, "", nil)
				}
				return agentID, err
			}
			agentAuth := auth.RequireAgentCredentialOrRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleOperator, auditedLookup, auditedCertificateLookup, isTokenRevoked)
			r.With(agentAuth).Post("/ingest", func(w http.ResponseWriter, r *http.Request) {
				var reading store.RFReading
				if err := json.NewDecoder(r.Body).Decode(&reading); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				reading.AgentID = strings.TrimSpace(reading.AgentID)

				if err := validate.AgentID(reading.AgentID); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.FrequencyHz(reading.FrequencyHz); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.PowerDbm(reading.PowerDbm); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.CapturedAt(reading.CapturedAt); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				if reading.SatelliteID != nil {
					exists, err := satellites.Exists(r.Context(), *reading.SatelliteID)
					if err != nil {
						http.Error(w, "failed to validate satellite", http.StatusInternalServerError)
						return
					}
					if !exists {
						http.Error(w, "satellite not found", http.StatusBadRequest)
						return
					}
				}

				if authenticatedAgentID, ok := auth.AgentIDFromContext(r.Context()); ok {
					if reading.AgentID != authenticatedAgentID {
						http.Error(w, "agent_id does not match the authenticated agent key", http.StatusForbidden)
						return
					}
				}

				history, err := rf.RecentForAgent(r.Context(), reading.AgentID, anomalyHistoryWindow)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := rf.Insert(r.Context(), &reading); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				detectRFAnomalies(r.Context(), rfAnomalies, detectionRules, incidents, auditLog, satellites, agentIdentities, spaceCVEs, tlePolicy(cfg), cfg.OrbitalAutoEscalation, reading, history)

				w.WriteHeader(http.StatusCreated)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/readings", func(w http.ResponseWriter, r *http.Request) {
				list, err := rf.Recent(r.Context(), 100)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/anomalies", func(w http.ResponseWriter, r *http.Request) {
				list, err := rfAnomalies.Recent(r.Context(), 100)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})
		})

		r.Route("/satellites", func(r chi.Router) {
			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				out, err := satellitesWithPositions(r.Context(), satellites, cache, tlePolicy(cfg))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(out)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/stream", func(w http.ResponseWriter, r *http.Request) {
				flusher, ok := w.(http.Flusher)
				if !ok {
					http.Error(w, "streaming unsupported", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("X-Accel-Buffering", "no")
				w.WriteHeader(http.StatusOK)
				metrics.SSEConnections.WithLabelValues("satellites").Inc()
				defer metrics.SSEConnections.WithLabelValues("satellites").Dec()

				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()

				sendUpdate := func() bool {
					out, err := satellitesWithPositions(r.Context(), satellites, cache, tlePolicy(cfg))
					if err != nil {
						return true
					}
					payload, err := json.Marshal(out)
					if err != nil {
						return true
					}
					if _, err := fmt.Fprintf(w, "event: satellites\ndata: %s\n\n", payload); err != nil {
						return false
					}
					flusher.Flush()
					return true
				}

				if !sendUpdate() {
					return
				}

				for {
					select {
					case <-r.Context().Done():
						return
					case <-ticker.C:
						if !sendUpdate() {
							return
						}
					}
				}
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleOperator, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					NoradID int    `json:"norad_id"`
					Name    string `json:"name"`
					Line1   string `json:"tle_line1"`
					Line2   string `json:"tle_line2"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Name = strings.TrimSpace(body.Name)

				if err := validate.NoradID(body.NoradID); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.SatelliteName(body.Name); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.TLELine(body.Line1, "1 "); err != nil {
					http.Error(w, "tle_line1: "+err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.TLELine(body.Line2, "2 "); err != nil {
					http.Error(w, "tle_line2: "+err.Error(), http.StatusBadRequest)
					return
				}

				if err := satellites.Upsert(r.Context(), body.NoradID, body.Name, body.Line1, body.Line2); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if claims, ok := auth.FromContext(r.Context()); ok {
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionSatelliteUpserted, clientIP(r), map[string]interface{}{"norad_id": body.NoradID, "name": body.Name})
				}
				w.WriteHeader(http.StatusCreated)
			})
		})

		r.Route("/tmtc", func(r chi.Router) {
			r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))

			r.Post("/decode/ax25", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					FrameHex string `json:"frame_hex"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if err := validate.AX25FrameHex(body.FrameHex); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				decoded, err := tmtc.DecodeAX25Hex(body.FrameHex)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(decoded)
			})

			r.Post("/decode/ccsds", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					PacketHex string `json:"packet_hex"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if err := validate.CCSDSPacketHex(body.PacketHex); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				decoded, err := tmtc.DecodeCCSDSHex(body.PacketHex)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(decoded)
			})

			r.Post("/decode/pus", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					PacketHex string `json:"packet_hex"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if err := validate.CCSDSPacketHex(body.PacketHex); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				decoded, err := tmtc.DecodePUSHex(body.PacketHex)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(decoded)
			})

			r.Post("/decode/csp", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					PacketHex string `json:"packet_hex"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if err := validate.CSPPacketHex(body.PacketHex); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				decoded, err := tmtc.DecodeCSPHex(body.PacketHex)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(decoded)
			})
		})

		r.With(auth.RequireAnyRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, []auth.Role{auth.RoleAuditor, auth.RoleAdmin}, isTokenRevoked)).Get("/audit-log", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)
			page, err := auditLog.List(r.Context(), limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/threat-intel/space-cves", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)
			filter := parseSpaceCVEFilter(r)

			cacheKey := fmt.Sprintf("lyrahcle:cache:space-cves:%d:%d:%s:%s:%s:%s:%s:%t",
				limit, offset, filter.Severity, filter.Keyword,
				filter.PublishedFrom.Format(time.RFC3339), filter.PublishedTo.Format(time.RFC3339),
				filter.SortBy, filter.SortDesc)
			var page store.Page[store.SpaceCVE]
			if hit, _ := cache.Get(r.Context(), cacheKey, &page); hit {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
				return
			}

			page, err := spaceCVEs.List(r.Context(), limit, offset, filter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cache.Set(r.Context(), cacheKey, page, 30*time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/threat-intel/space-cves/keywords", func(w http.ResponseWriter, r *http.Request) {
			const cacheKey = "lyrahcle:cache:space-cves:keywords"
			var keywords []string
			if hit, _ := cache.Get(r.Context(), cacheKey, &keywords); hit {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(keywords)
				return
			}

			keywords, err := spaceCVEs.DistinctKeywords(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cache.Set(r.Context(), cacheKey, keywords, 5*time.Minute)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(keywords)
		})

		r.Route("/incidents", func(r chi.Router) {
			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				limit, offset := pageParams(r)
				filter, err := parseIncidentFilter(r)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				page, err := incidents.List(r.Context(), limit, offset, filter)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/stream", func(w http.ResponseWriter, r *http.Request) {
				flusher, ok := w.(http.Flusher)
				if !ok {
					http.Error(w, "streaming unsupported", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("X-Accel-Buffering", "no")
				w.WriteHeader(http.StatusOK)
				metrics.SSEConnections.WithLabelValues("incidents").Inc()
				defer metrics.SSEConnections.WithLabelValues("incidents").Dec()

				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				send := func() bool {
					page, err := incidents.List(r.Context(), 50, 0, store.IncidentFilter{})
					if err != nil {
						return true
					}
					payload, err := json.Marshal(page)
					if err != nil {
						return true
					}
					if _, err := fmt.Fprintf(w, "event: incidents\ndata: %s\n\n", payload); err != nil {
						return false
					}
					flusher.Flush()
					return true
				}
				if !send() {
					return
				}
				for {
					select {
					case <-r.Context().Done():
						return
					case <-ticker.C:
						if !send() {
							return
						}
					}
				}
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleOperator, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Title       string `json:"title"`
					Description string `json:"description"`
					Severity    string `json:"severity"`
					SatelliteID string `json:"satellite_id"`
					AgentID     string `json:"agent_id"`
					AssignedTo  string `json:"assigned_to"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Title = strings.TrimSpace(body.Title)
				body.Description = strings.TrimSpace(body.Description)
				body.Severity = strings.ToLower(strings.TrimSpace(body.Severity))
				body.AgentID = strings.TrimSpace(body.AgentID)
				if err := validate.IncidentTitle(body.Title); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.IncidentDescription(body.Description); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.IncidentSeverity(body.Severity); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if body.AgentID != "" {
					if err := validate.AgentID(body.AgentID); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
				}

				satelliteID, err := optionalUUID(body.SatelliteID, "satellite_id")
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				assignedTo, err := optionalUUID(body.AssignedTo, "assigned_to")
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if satelliteID != nil {
					exists, err := satellites.Exists(r.Context(), *satelliteID)
					if err != nil {
						http.Error(w, "failed to validate satellite", http.StatusInternalServerError)
						return
					}
					if !exists {
						http.Error(w, "satellite not found", http.StatusBadRequest)
						return
					}
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if assignedTo != nil && claims.Role != auth.RoleAdmin {
					http.Error(w, "only admins can assign incidents", http.StatusForbidden)
					return
				}
				if assignedTo != nil {
					if _, err := users.GetByID(r.Context(), *assignedTo); err != nil {
						if err == store.ErrUserNotFound {
							http.Error(w, "assigned user not found", http.StatusBadRequest)
							return
						}
						http.Error(w, "failed to validate assigned user", http.StatusInternalServerError)
						return
					}
				}
				idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
				if idempotencyKey != "" && !validRequestID(idempotencyKey) {
					http.Error(w, "Idempotency-Key must contain only safe identifier characters and be at most 128 characters", http.StatusBadRequest)
					return
				}
				incident, err := incidents.Create(r.Context(), store.CreateIncidentInput{
					Title: body.Title, Description: body.Description, Severity: body.Severity, SourceType: "manual",
					SatelliteID: satelliteID, AgentID: body.AgentID, AssignedTo: assignedTo, CreatedBy: &claims.UserID, IdempotencyKey: idempotencyKey,
				})
				if err != nil {
					http.Error(w, "failed to create incident", http.StatusInternalServerError)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentCreated, clientIP(r), map[string]interface{}{"incident_id": incident.ID.String(), "reference": incident.Reference})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(incident)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/{incidentID}", func(w http.ResponseWriter, r *http.Request) {
				incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
				if err != nil {
					http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
					return
				}
				detail, err := incidents.Get(r.Context(), incidentID)
				if err != nil {
					if err == store.ErrIncidentNotFound {
						http.Error(w, "incident not found", http.StatusNotFound)
						return
					}
					http.Error(w, "failed to fetch incident", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(detail)
			})

			r.With(auth.RequireAnyRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, []auth.Role{auth.RoleAnalyst, auth.RoleOperator, auth.RoleAdmin}, isTokenRevoked)).Patch("/{incidentID}", func(w http.ResponseWriter, r *http.Request) {
				incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
				if err != nil {
					http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
					return
				}
				var body struct {
					Status     *string `json:"status"`
					Severity   *string `json:"severity"`
					AssignedTo *string `json:"assigned_to"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				if body.Status == nil && body.Severity == nil && body.AssignedTo == nil {
					http.Error(w, "at least one field is required", http.StatusBadRequest)
					return
				}

				input := store.UpdateIncidentInput{}
				if body.Status != nil {
					input.Status = strings.ToLower(strings.TrimSpace(*body.Status))
					if err := validate.IncidentStatus(input.Status); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
				}
				if body.Severity != nil {
					input.Severity = strings.ToLower(strings.TrimSpace(*body.Severity))
					if err := validate.IncidentSeverity(input.Severity); err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
				}
				if body.AssignedTo != nil {
					input.UpdateAssignment = true
					assignedTo, err := optionalUUID(*body.AssignedTo, "assigned_to")
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					input.AssignedTo = assignedTo
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if input.UpdateAssignment && claims.Role != auth.RoleAdmin {
					http.Error(w, "only admins can assign incidents", http.StatusForbidden)
					return
				}
				if input.AssignedTo != nil {
					if _, err := users.GetByID(r.Context(), *input.AssignedTo); err != nil {
						if err == store.ErrUserNotFound {
							http.Error(w, "assigned user not found", http.StatusBadRequest)
							return
						}
						http.Error(w, "failed to validate assigned user", http.StatusInternalServerError)
						return
					}
				}
				input.ActorID = claims.UserID
				incident, err := incidents.Update(r.Context(), incidentID, input)
				if err != nil {
					if errors.Is(err, store.ErrInvalidIncidentTransition) {
						http.Error(w, "invalid incident status transition", http.StatusConflict)
						return
					}
					if err == store.ErrIncidentNotFound {
						http.Error(w, "incident not found", http.StatusNotFound)
						return
					}
					http.Error(w, "failed to update incident", http.StatusInternalServerError)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentUpdated, clientIP(r), map[string]interface{}{"incident_id": incident.ID.String(), "reference": incident.Reference})
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(incident)
			})

			r.With(auth.RequireAnyRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, []auth.Role{auth.RoleAnalyst, auth.RoleOperator, auth.RoleAdmin}, isTokenRevoked)).Post("/{incidentID}/notes", func(w http.ResponseWriter, r *http.Request) {
				incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
				if err != nil {
					http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
					return
				}
				var body struct {
					Message string `json:"message"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Message = strings.TrimSpace(body.Message)
				if err := validate.IncidentNote(body.Message); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if err := incidents.AddNote(r.Context(), incidentID, claims.UserID, body.Message); err != nil {
					if err == store.ErrIncidentNotFound {
						http.Error(w, "incident not found", http.StatusNotFound)
						return
					}
					http.Error(w, "failed to add note", http.StatusInternalServerError)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentNoteAdded, clientIP(r), map[string]interface{}{"incident_id": incidentID.String()})
				w.WriteHeader(http.StatusNoContent)
			})

			r.With(auth.RequireAnyRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, []auth.Role{auth.RoleAnalyst, auth.RoleOperator, auth.RoleAdmin}, isTokenRevoked)).Post("/{incidentID}/evidence", func(w http.ResponseWriter, r *http.Request) {
				incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
				if err != nil {
					http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
					return
				}
				var body struct {
					EvidenceType string `json:"evidence_type"`
					EvidenceRef  string `json:"evidence_ref"`
					Label        string `json:"label"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.EvidenceType = strings.ToLower(strings.TrimSpace(body.EvidenceType))
				body.EvidenceRef = strings.TrimSpace(body.EvidenceRef)
				body.Label = strings.TrimSpace(body.Label)
				if err := validate.IncidentEvidenceType(body.EvidenceType); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.IncidentEvidenceRef(body.EvidenceRef); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if err := validate.IncidentEvidenceLabel(body.Label); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				evidence, err := incidents.AddEvidence(r.Context(), incidentID, store.CreateIncidentEvidenceInput{
					EvidenceType: body.EvidenceType, EvidenceRef: body.EvidenceRef, Label: body.Label, CreatedBy: &claims.UserID,
				})
				if err != nil {
					if errors.Is(err, store.ErrIncidentNotFound) {
						http.Error(w, "incident not found", http.StatusNotFound)
						return
					}
					http.Error(w, "failed to add incident evidence", http.StatusInternalServerError)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentEvidenceAdded, clientIP(r), map[string]interface{}{"incident_id": incidentID.String(), "evidence_type": body.EvidenceType})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(evidence)
			})
		})

		r.Route("/asset-alerts", func(r chi.Router) {
			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				limit, offset := pageParams(r)
				page, err := assetAlerts.List(r.Context(), limit, offset)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
			})

			r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleOperator, isTokenRevoked)).Post("/{alertID}/acknowledge", func(w http.ResponseWriter, r *http.Request) {
				alertID, err := uuid.Parse(chi.URLParam(r, "alertID"))
				if err != nil {
					http.Error(w, "alertID must be a valid UUID", http.StatusBadRequest)
					return
				}

				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				found, err := assetAlerts.Acknowledge(r.Context(), alertID, claims.UserID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if !found {
					http.Error(w, "alert not found", http.StatusNotFound)
					return
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAssetAlertAcked, clientIP(r), map[string]interface{}{"alert_id": alertID.String()})
				w.WriteHeader(http.StatusNoContent)
			})
		})
	})

	return r
}

func structuredRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		event := map[string]interface{}{
			"event":       "http_request",
			"method":      r.Method,
			"path":        r.URL.Path,
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

func detectRFAnomalies(ctx context.Context, anomalies *store.AnomalyStore, rules *store.DetectionRuleStore, incidents *store.IncidentStore, auditLog *store.AuditStore, satellites *store.SatelliteStore, identities *store.AgentIdentityStore, spaceCVEs *store.SpaceCVEStore, policy tlequality.Policy, orbitalAutoEscalation bool, reading store.RFReading, history []store.RFReading) {
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
		correlateRFDetectionRules(ctx, rules, incidents, auditLog, satellites, identities, spaceCVEs, policy, orbitalAutoEscalation, reading, created)
	}
}

func correlateRFDetectionRules(ctx context.Context, rules *store.DetectionRuleStore, incidents *store.IncidentStore, auditLog *store.AuditStore, satellites *store.SatelliteStore, identities *store.AgentIdentityStore, spaceCVEs *store.SpaceCVEStore, policy tlequality.Policy, orbitalAutoEscalation bool, reading store.RFReading, anomalyRecord store.RFAnomaly) {
	enabledRules, err := rules.ListEnabled(ctx)
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
		match, created, err := rules.RecordMatchIfAllowed(ctx, rule, anomalyRecord.ID, anomalyRecord.AgentID, anomalyRecord.Metric, anomalyRecord.ZScore)
		if err != nil {
			log.Printf("detection: failed to record rule %s match: %v", rule.ID, err)
			continue
		}
		if !created {
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
	incident, err := incidents.Create(ctx, store.CreateIncidentInput{
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
	if err := rules.AttachIncident(ctx, matchIDs, incident.ID); err != nil {
		log.Printf("detection: failed to link matches to incident %s: %v", incident.ID, err)
	}
	attachOrbitalSecurityEvidence(ctx, incidents, incident.ID.String(), orbitalSecurity)
	logAudit(ctx, auditLog, audit.ActorSystem, "correlation-engine", audit.ActionIncidentCreated, "", map[string]interface{}{
		"incident_id": incident.ID.String(),
		"reference":   incident.Reference,
		"anomaly_id":  anomalyRecord.ID.String(),
		"rule_count":  len(matched),
	})
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

func logAudit(ctx context.Context, auditLog *store.AuditStore, actorType, actorID, action, ip string, details map[string]interface{}) {
	if err := auditLog.Insert(ctx, actorType, actorID, action, ip, details); err != nil {
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
	const cacheKey = "lyrahcle:cache:satellites"
	var out []satWithPos
	if hit, _ := cache.Get(ctx, cacheKey, &out); hit {
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

	cache.Set(ctx, cacheKey, out, 5*time.Second)
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

	if _, err := users.CreateProvisioned(ctx, cfg.SeedAdminEmail, hash, string(auth.RoleAdmin)); err != nil {
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
