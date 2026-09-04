package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/satnogs"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tmtc"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (a *apiRoutes) registerSatelliteTMTCRoutes(r chi.Router) {
	auditLog := a.auditLog
	cache := a.cache
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	orbitalEvents := a.orbitalEvents
	satellites := a.satellites

	r.Route("/satellites", func(r chi.Router) {
		satnogsClient := satnogs.NewClient(10 * time.Second)
		r.With(a.perm("orbital.read")).Get("/{satelliteID}/external-sources", func(w http.ResponseWriter, r *http.Request) {
			satelliteID, err := uuid.Parse(chi.URLParam(r, "satelliteID"))
			if err != nil {
				http.Error(w, "invalid satellite ID", http.StatusBadRequest)
				return
			}
			satellite, err := satellites.Get(r.Context(), satelliteID)
			if err != nil {
				if err == pgx.ErrNoRows {
					http.Error(w, "satellite not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to load satellite", http.StatusInternalServerError)
				return
			}
			cacheKey := fmt.Sprintf("lyrahcle:cache:satnogs:%d", satellite.NoradID)
			var result satnogs.Intelligence
			if hit, _ := cache.Get(r.Context(), cacheKey, &result); !hit {
				result = satnogsClient.Fetch(r.Context(), satellite.NoradID)
				cache.Set(r.Context(), cacheKey, result, 15*time.Minute)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})
		r.With(a.perm("orbital.read")).Get("/{satelliteID}/external-observations", handleListExternalObservations(satellites, a.externalObservations))
		r.With(a.perm("orbital.manage")).Post("/{satelliteID}/external-observations/import", handleImportExternalObservations(satellites, a.externalObservations, satnogsClient, auditLog))
		r.With(a.perm("orbital.read")).Get("/{satelliteID}/passes", handleSatellitePassPrediction(satellites, tlePolicy(cfg)))
		r.With(a.perm("orbital.read")).Get("/{satelliteID}/orbital-events", handleListOrbitalEvents(orbitalEvents, satellites))
		r.With(a.perm("orbital.manage")).Post("/{satelliteID}/orbital-events", handleCreateOrbitalEvent(orbitalEvents, satellites, auditLog))
		r.With(a.perm("orbital.manage")).Put("/{satelliteID}", handleUpdateSatellite(satellites, auditLog, cache))

		r.With(a.perm("orbital.read")).Get("/", func(w http.ResponseWriter, r *http.Request) {
			out, err := satellitesWithPositions(r.Context(), satellites, cache, tlePolicy(cfg))
			if err != nil {
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(out)
		})

		r.With(a.perm("orbital.read")).Get("/stream", func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}
			if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
				log.Printf("satellites stream: disable write deadline: %v", err)
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			w.WriteHeader(http.StatusOK)
			metrics.SSEConnections.WithLabelValues("satellites").Inc()
			defer metrics.SSEConnections.WithLabelValues("satellites").Dec()

			updateTicker := time.NewTicker(5 * time.Second)
			defer updateTicker.Stop()
			heartbeatTicker := time.NewTicker(15 * time.Second)
			defer heartbeatTicker.Stop()
			var eventID uint64

			sendUpdate := func() bool {
				out, err := satellitesWithPositions(r.Context(), satellites, cache, tlePolicy(cfg))
				if err != nil {
					return true
				}
				payload, err := json.Marshal(out)
				if err != nil {
					return true
				}
				eventID++
				if _, err := fmt.Fprintf(w, "id: %d\nevent: satellites\ndata: %s\n\n", eventID, payload); err != nil {
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
				case <-updateTicker.C:
					if !sendUpdate() {
						return
					}
				case <-heartbeatTicker.C:
					if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
						return
					}
					flusher.Flush()
				}
			}
		})

		r.With(a.perm("orbital.manage")).Post("/", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				NoradID int    `json:"norad_id"`
				Name    string `json:"name"`
				Line1   string `json:"tle_line1"`
				Line2   string `json:"tle_line2"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
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
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
				return
			}
			if err := cache.Delete(r.Context(), satellitesCacheKey); err != nil {
				log.Printf("satellites: cache invalidation failed after upsert: %v", err)
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
			if err := decodeStrictJSON(r, &body); err != nil {
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
			if err := decodeStrictJSON(r, &body); err != nil {
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
			if err := decodeStrictJSON(r, &body); err != nil {
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

		r.Post("/analyze/ccsds-sequence", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				PacketHexes []string `json:"packet_hexes"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if len(body.PacketHexes) == 0 || len(body.PacketHexes) > 256 {
				http.Error(w, "packet_hexes must contain between 1 and 256 packets", http.StatusBadRequest)
				return
			}
			for _, packetHex := range body.PacketHexes {
				if err := validate.CCSDSPacketHex(packetHex); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			analysis, err := tmtc.AnalyzeCCSDSSequence(body.PacketHexes)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(analysis)
		})

		r.Post("/decode/csp", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				PacketHex string `json:"packet_hex"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
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
}
