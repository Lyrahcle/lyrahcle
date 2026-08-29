package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/alerts"
	"github.com/mlko01/lyrahcle/apps/backend/internal/anomaly"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/celestrak"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/db"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/orbit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/redisclient"
	"github.com/mlko01/lyrahcle/apps/backend/internal/respcache"
	"github.com/mlko01/lyrahcle/apps/backend/internal/scheduler"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tmtc"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tokenblocklist"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

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

	if err := seedAdmin(ctx, users, cfg); err != nil {
		log.Printf("seed admin: %v", err)
	}

	redisClient, err := redisclient.Connect(cfg.RedisURL, 5*time.Second)
	if err != nil {
		log.Printf("WARNING: Redis not available at %s (%v). Rate limiting will use an in-memory counter (does not survive a restart or work correctly with multiple replicas), and logout/password-change token revocation and response caching will be disabled.", cfg.RedisURL, err)
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
		syncTLEs(ctx, cfg.CelestrakGroups, satellites)
	})
	sched.Start()
	defer sched.Stop()

	go syncTLEs(ctx, cfg.CelestrakGroups, satellites)

	r := newRouter(cfg, assets, programs, rf, rfAnomalies, satellites, users, agentKeys, auditLog, spaceCVEs, assetAlerts, redisClient)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
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
	redisClient *redis.Client,
) http.Handler {
	blocklist := tokenblocklist.New(redisClient)
	cache := respcache.New(redisClient)
	isTokenRevoked := blocklist.IsRevoked

	globalLimiter := httprate.Limit(120, time.Minute, httprate.WithKeyByIP())
	authLimiter := httprate.Limit(10, time.Minute, httprate.WithKeyByIP())
	if redisClient != nil {
		globalLimiter = httprate.Limit(120, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient)))
		authLimiter = httprate.Limit(10, time.Minute, httprate.WithKeyByIP(), httprate.WithLimitCounter(ratelimit.NewRedisCounter(redisClient)))
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders(cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""))
	r.Use(metrics.Middleware)
	r.Use(maxBodySize(1 << 20))
	r.Use(globalLimiter)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"lyrahcle-backend"}`))
	})

	r.Get("/openapi.yaml", serveOpenAPISpec)
	r.Get("/docs", serveAPIDocs)

	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"message":"pong from lyrahcle"}`))
		})

		r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/system-info", func(w http.ResponseWriter, r *http.Request) {
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
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				limit, offset := pageParams(r)
				page, err := assets.List(r.Context(), limit, offset)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := programs.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
			r.Use(authLimiter)

			// Public self-registration is disabled: only an administrator can
			// create accounts, via POST /users below. The very first account
			// is created automatically at boot by seedAdmin (see SEED_ADMIN_EMAIL
			// / SEED_ADMIN_PASSWORD).

			r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
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
				if err != nil || !auth.CheckPassword(user.PasswordHash, body.Password) {
					logAudit(r.Context(), auditLog, audit.ActorAnonymous, body.Identifier, audit.ActionLoginFailed, clientIP(r), nil)
					http.Error(w, "invalid credentials", http.StatusUnauthorized)
					return
				}

				if user.TOTPEnabled {
					if body.TOTPCode == "" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						json.NewEncoder(w).Encode(map[string]string{"error": "totp_required"})
						return
					}
					if !auth.VerifyTOTP(user.TOTPSecret, body.TOTPCode) {
						logAudit(r.Context(), auditLog, audit.ActorAnonymous, body.Identifier, audit.ActionLoginFailed, clientIP(r), map[string]interface{}{"reason": "invalid_totp"})
						http.Error(w, "invalid two-factor code", http.StatusUnauthorized)
						return
					}
				}

				token, err := auth.IssueToken(cfg.JWTSecret, user.ID, auth.Role(user.Role), 24*time.Hour)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionLogin, clientIP(r), map[string]interface{}{"email": user.Email})

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"token": token, "role": user.Role})
			})

			r.Route("/totp", func(r chi.Router) {
				r.Use(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked))

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
					json.NewEncoder(w).Encode(map[string]string{"secret": secret, "otpauth_url": uri})
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
					if err := users.DisableTOTP(r.Context(), user.ID); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionTOTPDisabled, clientIP(r), nil)
					w.WriteHeader(http.StatusNoContent)
				})
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
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
					"id":           user.ID.String(),
					"email":        user.Email,
					"username":     user.Username,
					"role":         user.Role,
					"totp_enabled": user.TOTPEnabled,
				})
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Post("/logout", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				remaining := time.Until(claims.ExpiresAt.Time)
				if err := blocklist.Revoke(r.Context(), claims.ID, remaining); err != nil {
					log.Printf("logout: failed to revoke token for user %s: %v", claims.UserID, err)
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionLogout, clientIP(r), nil)
				w.WriteHeader(http.StatusNoContent)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Post("/change-password", func(w http.ResponseWriter, r *http.Request) {
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

				newHash, err := auth.HashPassword(body.NewPassword)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := users.UpdatePassword(r.Context(), user.ID, newHash); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// Force re-authentication with the new password: the token
				// used to make this request stays valid, by JWT signature
				// alone, until it would have expired anyway unless revoked.
				if claims, ok := auth.FromContext(r.Context()); ok {
					remaining := time.Until(claims.ExpiresAt.Time)
					if err := blocklist.Revoke(r.Context(), claims.ID, remaining); err != nil {
						log.Printf("change-password: failed to revoke token for user %s: %v", user.ID, err)
					}
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionPasswordChanged, clientIP(r), map[string]interface{}{"email": user.Email})

				w.WriteHeader(http.StatusNoContent)
			})
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(auth.RequireRole(cfg.JWTSecret, auth.RoleAdmin, isTokenRevoked))

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
					http.Error(w, "role must be one of: viewer, operator, admin", http.StatusBadRequest)
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
				json.NewEncoder(w).Encode(map[string]string{"id": user.ID.String(), "email": user.Email, "username": user.Username, "role": user.Role})
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
					http.Error(w, "role must be one of: viewer, operator, admin", http.StatusBadRequest)
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

				if target.Role == string(auth.RoleAdmin) && body.Role != string(auth.RoleAdmin) {
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

				updated, err := users.UpdateRole(r.Context(), userID, body.Role)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if claims, ok := auth.FromContext(r.Context()); ok {
					logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionRoleChanged, clientIP(r), map[string]interface{}{"target_user_id": updated.ID.String(), "target_email": updated.Email, "new_role": updated.Role})
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"id": updated.ID.String(), "email": updated.Email, "role": updated.Role})
			})
		})

		r.Route("/agent-keys", func(r chi.Router) {
			r.Use(auth.RequireRole(cfg.JWTSecret, auth.RoleAdmin, isTokenRevoked))

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

		r.Route("/rf", func(r chi.Router) {
			auditedLookup := func(ctx context.Context, keyHash string) (string, error) {
				agentID, err := agentKeys.AgentIDForHash(ctx, keyHash)
				if err != nil {
					logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentKeyAuthFailed, "", nil)
				}
				return agentID, err
			}
			agentAuth := auth.RequireAgentKeyOrRole(cfg.JWTSecret, auth.RoleOperator, auditedLookup, isTokenRevoked)
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

				detectRFAnomalies(r.Context(), rfAnomalies, reading, history)

				w.WriteHeader(http.StatusCreated)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/readings", func(w http.ResponseWriter, r *http.Request) {
				list, err := rf.Recent(r.Context(), 100)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/anomalies", func(w http.ResponseWriter, r *http.Request) {
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
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				type satWithPos struct {
					store.Satellite
					Position *orbit.Position `json:"position,omitempty"`
				}

				// Not user-scoped — every viewer+ sees the same list — so a
				// short TTL is safe. SGP4 propagation is cheap per call but
				// this is polled by every open dashboard every few seconds;
				// caching avoids redoing the same computation for each of them.
				const cacheKey = "lyrahcle:cache:satellites"
				var out []satWithPos
				if hit, _ := cache.Get(r.Context(), cacheKey, &out); hit {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(out)
					return
				}

				list, err := satellites.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				out = make([]satWithPos, 0, len(list))
				for _, sat := range list {
					entry := satWithPos{Satellite: sat}
					if pos, err := orbit.CurrentPosition(sat.TLELine1, sat.TLELine2); err == nil {
						entry.Position = &pos
					}
					out = append(out, entry)
				}

				cache.Set(r.Context(), cacheKey, out, 5*time.Second)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(out)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator, isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
			r.Use(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked))

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

		r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleAdmin, isTokenRevoked)).Get("/audit-log", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)
			page, err := auditLog.List(r.Context(), limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/threat-intel/space-cves", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)

			// The cve-sync worker only refreshes this feed once a day, so a
			// much longer TTL than the satellites cache is safe here.
			cacheKey := fmt.Sprintf("lyrahcle:cache:space-cves:%d:%d", limit, offset)
			var page store.Page[store.SpaceCVE]
			if hit, _ := cache.Get(r.Context(), cacheKey, &page); hit {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
				return
			}

			page, err := spaceCVEs.List(r.Context(), limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cache.Set(r.Context(), cacheKey, page, 30*time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.Route("/asset-alerts", func(r chi.Router) {
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer, isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				limit, offset := pageParams(r)
				page, err := assetAlerts.List(r.Context(), limit, offset)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator, isTokenRevoked)).Post("/{alertID}/acknowledge", func(w http.ResponseWriter, r *http.Request) {
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

func syncTLEs(ctx context.Context, groups []string, satellites *store.SatelliteStore) {
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}

		entries, err := celestrak.FetchGroup(group)
		if err != nil {
			log.Printf("tle-sync: fetch group %q: %v", group, err)
			continue
		}

		for _, e := range entries {
			if err := satellites.Upsert(ctx, e.NoradID, e.Name, e.Line1, e.Line2); err != nil {
				log.Printf("tle-sync: upsert %d (%s): %v", e.NoradID, e.Name, err)
			}
		}

		log.Printf("tle-sync: synced %d satellites from group %q", len(entries), group)
	}
}

const anomalyHistoryWindow = 20

func detectRFAnomalies(ctx context.Context, anomalies *store.AnomalyStore, reading store.RFReading, history []store.RFReading) {
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
		a := store.RFAnomaly{
			ReadingID:      reading.ID,
			AgentID:        reading.AgentID,
			Metric:         result.Metric,
			Value:          result.Value,
			BaselineMean:   result.Mean,
			BaselineStddev: result.StdDev,
			ZScore:         result.ZScore,
		}
		if err := anomalies.Insert(ctx, a); err != nil {
			log.Printf("anomaly: failed to record %s anomaly for agent %s: %v", result.Metric, reading.AgentID, err)
		}
	}
}

func logAudit(ctx context.Context, auditLog *store.AuditStore, actorType, actorID, action, ip string, details map[string]interface{}) {
	if err := auditLog.Insert(ctx, actorType, actorID, action, ip, details); err != nil {
		log.Printf("audit: failed to record %s by %s %s: %v", action, actorType, actorID, err)
	}
}

func clientIP(r *http.Request) string {
	if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); real != "" {
		return real
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
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

	if _, err := users.Create(ctx, cfg.SeedAdminEmail, hash, string(auth.RoleAdmin)); err != nil {
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
