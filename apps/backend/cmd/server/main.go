package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/celestrak"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/db"
	"github.com/mlko01/lyrahcle/apps/backend/internal/orbit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/scheduler"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tmtc"
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
	satellites := store.NewSatelliteStore(pool)
	users := store.NewUserStore(pool)
	agentKeys := store.NewAgentKeyStore(pool)
	auditLog := store.NewAuditStore(pool)
	spaceCVEs := store.NewSpaceCVEStore(pool)

	if err := seedAdmin(ctx, users, cfg); err != nil {
		log.Printf("seed admin: %v", err)
	}

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

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(maxBodySize(1 << 20))
	r.Use(httprate.LimitByIP(120, time.Minute))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"lyrahcle-backend"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"message":"pong from lyrahcle"}`))
		})

		r.Route("/assets", func(r chi.Router) {
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := assets.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator)).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := programs.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator)).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
			r.Use(httprate.LimitByIP(10, time.Minute))

			r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Email    string `json:"email"`
					Password string `json:"password"`
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

				count, err := users.Count(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				role := string(auth.RoleViewer)
				if count == 0 {
					role = string(auth.RoleAdmin)
				}

				hash, err := auth.HashPassword(body.Password)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				user, err := users.Create(r.Context(), body.Email, hash, role)
				if err != nil {
					if err == store.ErrEmailTaken {
						http.Error(w, "email already registered", http.StatusConflict)
						return
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionRegister, clientIP(r), map[string]interface{}{"email": user.Email, "role": user.Role})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]string{"id": user.ID.String(), "email": user.Email, "role": user.Role})
			})

			r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Email    string `json:"email"`
					Password string `json:"password"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				body.Email = strings.TrimSpace(strings.ToLower(body.Email))
				if body.Email == "" || body.Password == "" {
					http.Error(w, "invalid email or password", http.StatusUnauthorized)
					return
				}

				user, err := users.GetByEmail(r.Context(), body.Email)
				if err != nil || !auth.CheckPassword(user.PasswordHash, body.Password) {
					logAudit(r.Context(), auditLog, audit.ActorAnonymous, body.Email, audit.ActionLoginFailed, clientIP(r), nil)
					http.Error(w, "invalid email or password", http.StatusUnauthorized)
					return
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

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer)).Post("/change-password", func(w http.ResponseWriter, r *http.Request) {
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

				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionPasswordChanged, clientIP(r), map[string]interface{}{"email": user.Email})

				w.WriteHeader(http.StatusNoContent)
			})
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(auth.RequireRole(cfg.JWTSecret, auth.RoleAdmin))

			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := users.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
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
			r.Use(auth.RequireRole(cfg.JWTSecret, auth.RoleAdmin))

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
			agentAuth := auth.RequireAgentKeyOrRole(cfg.JWTSecret, auth.RoleOperator, auditedLookup)
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

				if err := rf.Insert(r.Context(), reading); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer)).Get("/readings", func(w http.ResponseWriter, r *http.Request) {
				list, err := rf.Recent(r.Context(), 100)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
			})
		})

		r.Route("/satellites", func(r chi.Router) {
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer)).Get("/", func(w http.ResponseWriter, r *http.Request) {
				list, err := satellites.List(r.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				type satWithPos struct {
					store.Satellite
					Position *orbit.Position `json:"position,omitempty"`
				}

				out := make([]satWithPos, 0, len(list))
				for _, sat := range list {
					entry := satWithPos{Satellite: sat}
					if pos, err := orbit.CurrentPosition(sat.TLELine1, sat.TLELine2); err == nil {
						entry.Position = &pos
					}
					out = append(out, entry)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(out)
			})

			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator)).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
		})

		r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleAdmin)).Get("/audit-log", func(w http.ResponseWriter, r *http.Request) {
			entries, err := auditLog.List(r.Context(), 200)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entries)
		})

		r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleViewer)).Get("/threat-intel/space-cves", func(w http.ResponseWriter, r *http.Request) {
			cves, err := spaceCVEs.List(r.Context(), 200)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cves)
		})
	})

	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		log.Printf("lyrahcle backend listening on :%s (TLS enabled)", cfg.Port)
		if err := http.ListenAndServeTLS(":"+cfg.Port, cfg.TLSCertFile, cfg.TLSKeyFile, r); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Printf("lyrahcle backend listening on :%s (plain HTTP)", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}

func maxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
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
