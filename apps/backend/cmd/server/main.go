package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

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
	rf := store.NewRFStore(pool)
	satellites := store.NewSatelliteStore(pool)
	users := store.NewUserStore(pool)

	sched := scheduler.New()
	_ = sched.Every("0 */6 * * *", "recon-sweep", func() {
		log.Println("recon sweep tick (not yet implemented)")
	})
	_ = sched.Every("0 */6 * * *", "tle-sync", func() {
		syncTLEs(ctx, cfg.CelestrakGroups, satellites)
	})
	sched.Start()
	defer sched.Stop()

	syncTLEs(ctx, cfg.CelestrakGroups, satellites)

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

		r.Get("/assets", func(w http.ResponseWriter, r *http.Request) {
			list, err := assets.List(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
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
					http.Error(w, "invalid email or password", http.StatusUnauthorized)
					return
				}

				token, err := auth.IssueToken(cfg.JWTSecret, user.ID, auth.Role(user.Role), 24*time.Hour)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"token": token, "role": user.Role})
			})
		})

		r.Route("/rf", func(r chi.Router) {
			r.With(auth.RequireRole(cfg.JWTSecret, auth.RoleOperator)).Post("/ingest", func(w http.ResponseWriter, r *http.Request) {
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

				if err := rf.Insert(r.Context(), reading); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)
			})

			r.Get("/readings", func(w http.ResponseWriter, r *http.Request) {
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
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
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
