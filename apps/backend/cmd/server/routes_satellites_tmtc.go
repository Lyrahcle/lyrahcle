package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tmtc"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

func (a *apiRoutes) registerSatelliteTMTCRoutes(r chi.Router) {
	auditLog := a.auditLog
	cache := a.cache
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	orbitalEvents := a.orbitalEvents
	rbac := a.rbac
	satellites := a.satellites

	r.Route("/satellites", func(r chi.Router) {
		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.read", isTokenRevoked)).Get("/{satelliteID}/passes", handleSatellitePassPrediction(satellites, tlePolicy(cfg)))
		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.read", isTokenRevoked)).Get("/{satelliteID}/orbital-events", handleListOrbitalEvents(orbitalEvents, satellites))
		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.manage", isTokenRevoked)).Post("/{satelliteID}/orbital-events", handleCreateOrbitalEvent(orbitalEvents, satellites, auditLog))
		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.manage", isTokenRevoked)).Put("/{satelliteID}", handleUpdateSatellite(satellites, auditLog, cache))

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.read", isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
			out, err := satellitesWithPositions(r.Context(), satellites, cache, tlePolicy(cfg))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(out)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.read", isTokenRevoked)).Get("/stream", func(w http.ResponseWriter, r *http.Request) {
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

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.manage", isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
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
