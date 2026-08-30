package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/respcache"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

const satellitesCacheKey = "lyrahcle:cache:satellites"

type updateSatelliteRequest struct {
	NoradID int    `json:"norad_id"`
	Name    string `json:"name"`
	Line1   string `json:"tle_line1"`
	Line2   string `json:"tle_line2"`
}

func handleUpdateSatellite(satellites *store.SatelliteStore, auditLog *store.AuditStore, cache *respcache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		satelliteID, err := uuid.Parse(chi.URLParam(r, "satelliteID"))
		if err != nil {
			http.Error(w, "satelliteID must be a valid UUID", http.StatusBadRequest)
			return
		}

		var body updateSatelliteRequest
		if err := decodeStrictJSON(r, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		body.Name = strings.TrimSpace(body.Name)
		body.Line1 = strings.TrimSpace(body.Line1)
		body.Line2 = strings.TrimSpace(body.Line2)

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

		satellite, err := satellites.Update(r.Context(), satelliteID, body.NoradID, body.Name, body.Line1, body.Line2)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "satellite not found", http.StatusNotFound)
				return
			}
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				http.Error(w, "NORAD ID is already registered", http.StatusConflict)
				return
			}
			http.Error(w, "failed to update satellite", http.StatusInternalServerError)
			return
		}

		if err := cache.Delete(r.Context(), satellitesCacheKey); err != nil {
			log.Printf("satellites: cache invalidation failed after update: %v", err)
		}
		if claims, ok := auth.FromContext(r.Context()); ok {
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionSatelliteUpdated, clientIP(r), map[string]interface{}{
				"satellite_id": satellite.ID.String(),
				"norad_id":     satellite.NoradID,
				"name":         satellite.Name,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(satellite); err != nil {
			log.Printf("satellites: failed to encode update response: %v", err)
		}
	}
}
