package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/satnogs"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

const (
	externalObservationDefaultDays = 30
	externalObservationMaxDays     = 90
	externalObservationListLimit   = 50
)

func handleListExternalObservations(satellites *store.SatelliteStore, observations *store.ExternalObservationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		satellite, ok := loadSatelliteForRequest(w, r, satellites)
		if !ok {
			return
		}
		stored, err := observations.ListBySatellite(r.Context(), orgID, satellite.ID, externalObservationListLimit)
		if err != nil {
			http.Error(w, "failed to load external observations", http.StatusInternalServerError)
			return
		}
		records := make([]satnogs.ObservationRecord, 0, len(stored))
		for _, observation := range stored {
			records = append(records, observationRecordFromStore(observation))
		}
		payload := struct {
			SatelliteID   uuid.UUID                   `json:"satellite_id"`
			NORADID       int                         `json:"norad_id"`
			EvidenceKind  string                      `json:"evidence_kind"`
			Applicability string                      `json:"applicability"`
			Warning       string                      `json:"warning"`
			Observations  []store.ExternalObservation `json:"observations"`
			Geometry      []satnogs.GeometryCheck     `json:"geometry"`
		}{
			SatelliteID:   satellite.ID,
			NORADID:       satellite.NoradID,
			EvidenceKind:  satnogs.EvidenceKindExternal,
			Applicability: satnogs.ApplicabilityReception,
			Warning:       "Community ground reception proves the downlink was observable by a third party. It does not prove onboard health, and its absence does not prove an outage.",
			Observations:  stored,
			Geometry:      satnogs.CheckGeometry(records, satellite.TLELine1, satellite.TLELine2),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func handleImportExternalObservations(satellites *store.SatelliteStore, observations *store.ExternalObservationStore, client *satnogs.Client, auditLog *store.AuditStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		claims, ok := auth.FromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		satellite, ok := loadSatelliteForRequest(w, r, satellites)
		if !ok {
			return
		}

		var body struct {
			WindowDays int `json:"window_days"`
			Limit      int `json:"limit"`
		}
		if r.ContentLength > 0 {
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
		}
		windowDays := body.WindowDays
		if windowDays <= 0 {
			windowDays = externalObservationDefaultDays
		}
		if windowDays > externalObservationMaxDays {
			windowDays = externalObservationMaxDays
		}
		limit := body.Limit
		if limit <= 0 || limit > satnogs.MaxImportRecords {
			limit = satnogs.MaxImportRecords
		}

		result := client.Import(r.Context(), satellite.NoradID, time.Now().UTC().AddDate(0, 0, -windowDays), limit)
		if !result.SourceState.Available {
			http.Error(w, "external source unavailable", http.StatusBadGateway)
			return
		}

		inputs := make([]store.ExternalObservationInput, 0, len(result.Records))
		for _, record := range result.Records {
			inputs = append(inputs, storeInputFromRecord(record))
		}
		stored, err := observations.UpsertBatch(r.Context(), orgID, satellite.ID, inputs)
		if err != nil {
			http.Error(w, "failed to store external observations", http.StatusInternalServerError)
			return
		}

		logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionExternalReceptionImported, clientIP(r), map[string]interface{}{
			"organization_id": orgID.String(),
			"satellite_id":    satellite.ID.String(),
			"norad_id":        satellite.NoradID,
			"source":          result.Source,
			"stored":          stored,
			"skipped":         result.Skipped,
			"window_days":     windowDays,
		})

		payload := struct {
			SatelliteID   uuid.UUID               `json:"satellite_id"`
			NORADID       int                     `json:"norad_id"`
			Source        string                  `json:"source"`
			WindowDays    int                     `json:"window_days"`
			Stored        int                     `json:"stored"`
			Skipped       int                     `json:"skipped"`
			EvidenceKind  string                  `json:"evidence_kind"`
			Applicability string                  `json:"applicability"`
			SourceState   satnogs.SourceState     `json:"source_state"`
			Geometry      []satnogs.GeometryCheck `json:"geometry"`
		}{
			SatelliteID:   satellite.ID,
			NORADID:       satellite.NoradID,
			Source:        result.Source,
			WindowDays:    windowDays,
			Stored:        stored,
			Skipped:       result.Skipped,
			EvidenceKind:  satnogs.EvidenceKindExternal,
			Applicability: satnogs.ApplicabilityReception,
			SourceState:   result.SourceState,
			Geometry:      satnogs.CheckGeometry(result.Records, satellite.TLELine1, satellite.TLELine2),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func loadSatelliteForRequest(w http.ResponseWriter, r *http.Request, satellites *store.SatelliteStore) (store.Satellite, bool) {
	satelliteID, err := uuid.Parse(chi.URLParam(r, "satelliteID"))
	if err != nil {
		http.Error(w, "invalid satellite ID", http.StatusBadRequest)
		return store.Satellite{}, false
	}
	satellite, err := satellites.Get(r.Context(), satelliteID)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "satellite not found", http.StatusNotFound)
			return store.Satellite{}, false
		}
		http.Error(w, "failed to load satellite", http.StatusInternalServerError)
		return store.Satellite{}, false
	}
	return satellite, true
}

func storeInputFromRecord(record satnogs.ObservationRecord) store.ExternalObservationInput {
	return store.ExternalObservationInput{
		Source:             record.Source,
		ExternalID:         record.ExternalID,
		StationID:          record.StationID,
		StationName:        record.StationName,
		Observer:           record.Observer,
		TransmitterUUID:    record.TransmitterUUID,
		Mode:               record.Mode,
		DownlinkHz:         record.DownlinkHz,
		StationLatitude:    record.StationLatitude,
		StationLongitude:   record.StationLongitude,
		StartedAt:          record.StartedAt,
		EndedAt:            record.EndedAt,
		Status:             record.Status,
		VettedStatus:       record.VettedStatus,
		ReceptionConfirmed: record.ReceptionConfirmed,
		EvidenceKind:       record.EvidenceKind,
		Applicability:      record.Applicability,
		Confidence:         record.Confidence,
		WaterfallURL:       record.WaterfallURL,
	}
}

func observationRecordFromStore(observation store.ExternalObservation) satnogs.ObservationRecord {
	return satnogs.ObservationRecord{
		ExternalID:         observation.ExternalID,
		Source:             observation.Source,
		StationID:          observation.StationID,
		StationName:        observation.StationName,
		Observer:           observation.Observer,
		TransmitterUUID:    observation.TransmitterUUID,
		Mode:               observation.Mode,
		DownlinkHz:         observation.DownlinkHz,
		StationLatitude:    observation.StationLatitude,
		StationLongitude:   observation.StationLongitude,
		StartedAt:          observation.StartedAt,
		EndedAt:            observation.EndedAt,
		Status:             observation.Status,
		VettedStatus:       observation.VettedStatus,
		ReceptionConfirmed: observation.ReceptionConfirmed,
		EvidenceKind:       observation.EvidenceKind,
		Applicability:      observation.Applicability,
		Confidence:         observation.Confidence,
		ObservedAt:         observation.EndedAt,
		WaterfallURL:       observation.WaterfallURL,
	}
}
