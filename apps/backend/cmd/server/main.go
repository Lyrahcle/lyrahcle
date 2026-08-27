package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mlko01/lyrahcle/apps/backend/internal/celestrak"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/db"
	"github.com/mlko01/lyrahcle/apps/backend/internal/orbit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/scheduler"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tmtc"
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

		r.Route("/rf", func(r chi.Router) {
			r.Post("/ingest", func(w http.ResponseWriter, r *http.Request) {
				var reading store.RFReading
				if err := json.NewDecoder(r.Body).Decode(&reading); err != nil {
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

			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					NoradID int    `json:"norad_id"`
					Name    string `json:"name"`
					Line1   string `json:"tle_line1"`
					Line2   string `json:"tle_line2"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
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

	log.Printf("lyrahcle backend listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
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
