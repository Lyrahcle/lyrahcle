package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func handleListOrbitalEvents(events *store.OrbitalEventStore, satellites *store.SatelliteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		satelliteID, err := uuid.Parse(chi.URLParam(r, "satelliteID"))
		if err != nil {
			http.Error(w, "invalid satelliteID", http.StatusBadRequest)
			return
		}
		exists, err := satellites.Exists(r.Context(), satelliteID)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
			return
		}
		if !exists {
			http.Error(w, "satellite not found", http.StatusNotFound)
			return
		}

		hours := 168.0
		if raw := strings.TrimSpace(r.URL.Query().Get("hours")); raw != "" {
			hours, err = strconv.ParseFloat(raw, 64)
			if err != nil || hours <= 0 || hours > 24*90 {
				http.Error(w, "hours must be > 0 and <= 2160", http.StatusBadRequest)
				return
			}
		}
		limit := 50
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			limit, err = strconv.Atoi(raw)
			if err != nil || limit < 1 || limit > 200 {
				http.Error(w, "limit must be between 1 and 200", http.StatusBadRequest)
				return
			}
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		list, err := events.RecentForSatelliteInOrganization(r.Context(), orgID, satelliteID, time.Now().UTC().Add(-time.Duration(hours*float64(time.Hour))), limit)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	}
}

func handleCreateOrbitalEvent(events *store.OrbitalEventStore, satellites *store.SatelliteStore, auditLog *store.AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		satelliteID, err := uuid.Parse(chi.URLParam(r, "satelliteID"))
		if err != nil {
			http.Error(w, "invalid satelliteID", http.StatusBadRequest)
			return
		}
		exists, err := satellites.Exists(r.Context(), satelliteID)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
			return
		}
		if !exists {
			http.Error(w, "satellite not found", http.StatusNotFound)
			return
		}

		var body struct {
			EventType  string                 `json:"event_type"`
			EventTime  time.Time              `json:"event_time"`
			Source     string                 `json:"source"`
			Confidence float64                `json:"confidence"`
			Severity   string                 `json:"severity"`
			Metadata   map[string]interface{} `json:"metadata"`
		}
		if err := decodeStrictJSON(r, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		body.EventType = strings.TrimSpace(strings.ToLower(body.EventType))
		body.Source = strings.TrimSpace(body.Source)
		body.Severity = strings.TrimSpace(strings.ToLower(body.Severity))
		if body.EventType != "maneuver" && body.EventType != "conjunction" {
			http.Error(w, "event_type must be maneuver or conjunction", http.StatusBadRequest)
			return
		}
		if body.EventTime.IsZero() {
			http.Error(w, "event_time is required", http.StatusBadRequest)
			return
		}
		if body.EventTime.Before(time.Now().UTC().Add(-365*24*time.Hour)) || body.EventTime.After(time.Now().UTC().Add(365*24*time.Hour)) {
			http.Error(w, "event_time must be within one year of now", http.StatusBadRequest)
			return
		}
		if body.Source == "" || len(body.Source) > 160 {
			http.Error(w, "source must be between 1 and 160 characters", http.StatusBadRequest)
			return
		}
		if body.Confidence < 0 || body.Confidence > 1 {
			http.Error(w, "confidence must be between 0 and 1", http.StatusBadRequest)
			return
		}
		switch body.Severity {
		case "info", "low", "medium", "high", "critical":
		default:
			http.Error(w, "severity must be one of: info, low, medium, high, critical", http.StatusBadRequest)
			return
		}
		if body.Metadata == nil {
			body.Metadata = map[string]interface{}{}
		}
		body.Metadata["candidate_only"] = true
		body.Metadata["ingestion_mode"] = "operator_or_trusted_integration"

		var createdBy *uuid.UUID
		if claims, ok := auth.FromContext(r.Context()); ok {
			id := claims.UserID
			createdBy = &id
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		created, err := events.Create(r.Context(), store.CreateOrbitalEventInput{
			OrganizationID: orgID, SatelliteID: satelliteID, EventType: body.EventType, EventTime: body.EventTime.UTC(), Source: body.Source,
			Confidence: body.Confidence, Severity: body.Severity, Metadata: body.Metadata, CreatedBy: createdBy,
		})
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
			return
		}
		if createdBy != nil {
			logAudit(r.Context(), auditLog, audit.ActorUser, createdBy.String(), audit.ActionOrbitalEventCreated, clientIP(r), map[string]interface{}{
				"orbital_event_id": created.ID.String(), "satellite_id": satelliteID.String(), "event_type": created.EventType,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)
	}
}
