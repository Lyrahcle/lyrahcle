package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mlko01/lyrahcle/apps/backend/internal/sparta"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

const (
	spartaMaxEvidenceWindowDays = 90
	spartaDefaultWindowDays     = 30
	spartaEvidenceLimit         = 100
)

func (a *apiRoutes) registerSpartaRoutes(r chi.Router) {
	r.Route("/sparta", func(r chi.Router) {
		r.With(a.perm("detections.read")).Get("/matrix", handleSpartaMatrix())
		r.With(a.perm("detections.read")).Get("/coverage", a.handleSpartaCoverage())
	})
}

func handleSpartaMatrix() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload := struct {
			Framework        string             `json:"framework"`
			FrameworkOwner   string             `json:"framework_owner"`
			FrameworkURL     string             `json:"framework_url"`
			CatalogVersion   string             `json:"catalog_version"`
			CatalogScopeNote string             `json:"catalog_scope_note"`
			Tactics          []sparta.Tactic    `json:"tactics"`
			Techniques       []sparta.Technique `json:"techniques"`
			Rules            []sparta.Rule      `json:"rules"`
		}{
			Framework:        sparta.FrameworkName,
			FrameworkOwner:   sparta.FrameworkOwner,
			FrameworkURL:     sparta.FrameworkURL,
			CatalogVersion:   sparta.CatalogVersion,
			CatalogScopeNote: sparta.CatalogScopeNote,
			Tactics:          sparta.Tactics(),
			Techniques:       sparta.Techniques(),
			Rules:            sparta.Rules(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func (a *apiRoutes) handleSpartaCoverage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		windowDays := spartaWindowDays(r.URL.Query().Get("window_days"))
		now := time.Now().UTC()
		since := now.AddDate(0, 0, -windowDays)

		observations := make([]sparta.ObservedAssessment, 0, spartaEvidenceLimit)

		anomalies, err := a.rfAnomalies.RecentByOrganization(r.Context(), orgID, spartaEvidenceLimit)
		if err != nil {
			http.Error(w, "failed to load RF anomalies", http.StatusInternalServerError)
			return
		}
		for _, anomaly := range anomalies {
			if anomaly.DetectedAt.Before(since) {
				continue
			}
			for _, assessment := range sparta.AssessAnomaly(anomaly.Metric, anomaly.ZScore, anomaly.ID.String()) {
				observations = append(observations, sparta.ObservedAssessment{Assessment: assessment, ObservedAt: anomaly.DetectedAt, EvidenceRef: anomaly.ID.String()})
			}
		}

		incidents, err := a.incidents.ListByOrganization(r.Context(), orgID, spartaEvidenceLimit, 0, store.IncidentFilter{})
		if err != nil {
			http.Error(w, "failed to load incidents", http.StatusInternalServerError)
			return
		}
		for _, incident := range incidents.Items {
			if incident.CreatedAt.Before(since) {
				continue
			}
			for _, assessment := range sparta.AssessIncident(incident.SourceType, incident.ID.String()) {
				observations = append(observations, sparta.ObservedAssessment{Assessment: assessment, ObservedAt: incident.CreatedAt, EvidenceRef: incident.ID.String()})
			}
		}

		if a.externalObservations != nil {
			external, err := a.externalObservations.RecentByOrganization(r.Context(), orgID, spartaEvidenceLimit)
			if err != nil {
				http.Error(w, "failed to load external observations", http.StatusInternalServerError)
				return
			}
			for _, observation := range external {
				if observation.EndedAt.Before(since) {
					continue
				}
				reference := observation.ID.String()
				for _, assessment := range sparta.Assess(sparta.EvidenceExternalRecep, "community_observation", reference) {
					observations = append(observations, sparta.ObservedAssessment{Assessment: assessment, ObservedAt: observation.EndedAt, EvidenceRef: reference})
				}
			}
		}

		matrix := sparta.BuildMatrix(observations, now)
		payload := struct {
			sparta.Matrix
			WindowDays int       `json:"window_days"`
			WindowFrom time.Time `json:"window_from"`
			Note       string    `json:"note"`
		}{
			Matrix:     matrix,
			WindowDays: windowDays,
			WindowFrom: since,
			Note:       "Coverage reflects evidence Lyrahcle has actually observed in this window. Context-only evidence never establishes that a technique was executed.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func spartaWindowDays(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return spartaDefaultWindowDays
	}
	if value > spartaMaxEvidenceWindowDays {
		return spartaMaxEvidenceWindowDays
	}
	return value
}
