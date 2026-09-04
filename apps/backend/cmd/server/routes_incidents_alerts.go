package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	incidentdomain "github.com/mlko01/lyrahcle/apps/backend/internal/incident"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

func (a *apiRoutes) registerIncidentAlertRoutes(r chi.Router) {
	assetAlerts := a.assetAlerts
	auditLog := a.auditLog
	cache := a.cache
	incidentService := a.incidentService
	rbac := a.rbac
	satellites := a.satellites
	users := a.users

	r.Route("/incidents", func(r chi.Router) {
		r.With(a.perm("incidents.read")).Get("/", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)
			filter, err := parseIncidentFilter(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			if !requireRequestPermission(w, r, rbac, "incidents.read") {
				return
			}
			cacheKey := incidentsCacheKey(orgID, limit, offset, filter)
			var page store.Page[store.Incident]
			if hit, _ := cache.Get(r.Context(), cacheKey, &page); hit {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
				return
			}
			page, err = incidentService.List(r.Context(), orgID, limit, offset, filter)
			if err != nil {
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
				return
			}
			cache.Set(r.Context(), cacheKey, page, 3*time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.With(a.perm("incidents.read")).Get("/stream", func(w http.ResponseWriter, r *http.Request) {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}
			if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
				log.Printf("incidents stream: disable write deadline: %v", err)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			w.WriteHeader(http.StatusOK)
			metrics.SSEConnections.WithLabelValues("incidents").Inc()
			defer metrics.SSEConnections.WithLabelValues("incidents").Dec()

			updateTicker := time.NewTicker(5 * time.Second)
			defer updateTicker.Stop()
			heartbeatTicker := time.NewTicker(15 * time.Second)
			defer heartbeatTicker.Stop()
			var eventID uint64
			send := func() bool {
				orgID, ok := requestOrganizationID(w, r)
				if !ok {
					return false
				}
				page, err := incidentService.List(r.Context(), orgID, 50, 0, store.IncidentFilter{})
				if err != nil {
					return true
				}
				payload, err := json.Marshal(page)
				if err != nil {
					return true
				}
				eventID++
				if _, err := fmt.Fprintf(w, "id: %d\nevent: incidents\ndata: %s\n\n", eventID, payload); err != nil {
					return false
				}
				flusher.Flush()
				return true
			}
			if !send() {
				return
			}
			for {
				select {
				case <-r.Context().Done():
					return
				case <-updateTicker.C:
					if !send() {
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

		r.With(a.perm("incidents.manage")).Post("/", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Severity    string `json:"severity"`
				SatelliteID string `json:"satellite_id"`
				AgentID     string `json:"agent_id"`
				AssignedTo  string `json:"assigned_to"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Title = strings.TrimSpace(body.Title)
			body.Description = strings.TrimSpace(body.Description)
			body.Severity = strings.ToLower(strings.TrimSpace(body.Severity))
			body.AgentID = strings.TrimSpace(body.AgentID)
			if err := validate.IncidentTitle(body.Title); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.IncidentDescription(body.Description); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.IncidentSeverity(body.Severity); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.AgentID != "" {
				if err := validate.AgentID(body.AgentID); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}

			satelliteID, err := optionalUUID(body.SatelliteID, "satellite_id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assignedTo, err := optionalUUID(body.AssignedTo, "assigned_to")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if satelliteID != nil {
				exists, err := satellites.Exists(r.Context(), *satelliteID)
				if err != nil {
					http.Error(w, "failed to validate satellite", http.StatusInternalServerError)
					return
				}
				if !exists {
					http.Error(w, "satellite not found", http.StatusBadRequest)
					return
				}
			}
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			orgID, err := auth.OrganizationID(claims)
			if err != nil {
				http.Error(w, "organization context required", http.StatusForbidden)
				return
			}
			if assignedTo != nil {
				if _, err := users.GetByIDInOrganization(r.Context(), orgID, *assignedTo); err != nil {
					if err == store.ErrUserNotFound {
						http.Error(w, "assigned user not found", http.StatusBadRequest)
						return
					}
					http.Error(w, "failed to validate assigned user", http.StatusInternalServerError)
					return
				}
			}
			idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if idempotencyKey != "" && !validRequestID(idempotencyKey) {
				http.Error(w, "Idempotency-Key must contain only safe identifier characters and be at most 128 characters", http.StatusBadRequest)
				return
			}
			incident, err := incidentService.Create(r.Context(), incidentdomain.CreateInput{
				OrganizationID: orgID, ActorID: claims.UserID, Title: body.Title, Description: body.Description, Severity: body.Severity,
				SatelliteID: satelliteID, AgentID: body.AgentID, AssignedTo: assignedTo, IdempotencyKey: idempotencyKey,
			})
			if err != nil {
				http.Error(w, "failed to create incident", http.StatusInternalServerError)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentCreated, clientIP(r), map[string]interface{}{"incident_id": incident.ID.String(), "reference": incident.Reference})
			if err := cache.Delete(r.Context(), incidentsCacheKeyDefault(orgID)); err != nil {
				log.Printf("incidents: cache invalidation failed after create: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(incident)
		})

		r.With(a.perm("incidents.read")).Get("/{incidentID}", func(w http.ResponseWriter, r *http.Request) {
			incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
			if err != nil {
				http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			detail, err := incidentService.Get(r.Context(), orgID, incidentID)
			if err != nil {
				if err == store.ErrIncidentNotFound {
					http.Error(w, "incident not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to fetch incident", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(detail)
		})

		r.With(a.perm("incidents.manage")).Patch("/{incidentID}", func(w http.ResponseWriter, r *http.Request) {
			incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
			if err != nil {
				http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
				return
			}
			var body struct {
				Status     *string `json:"status"`
				Severity   *string `json:"severity"`
				AssignedTo *string `json:"assigned_to"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if body.Status == nil && body.Severity == nil && body.AssignedTo == nil {
				http.Error(w, "at least one field is required", http.StatusBadRequest)
				return
			}

			input := store.UpdateIncidentInput{}
			if body.Status != nil {
				input.Status = strings.ToLower(strings.TrimSpace(*body.Status))
				if err := validate.IncidentStatus(input.Status); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			if body.Severity != nil {
				input.Severity = strings.ToLower(strings.TrimSpace(*body.Severity))
				if err := validate.IncidentSeverity(input.Severity); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			if body.AssignedTo != nil {
				input.UpdateAssignment = true
				assignedTo, err := optionalUUID(*body.AssignedTo, "assigned_to")
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				input.AssignedTo = assignedTo
			}
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			orgID, err := auth.OrganizationID(claims)
			if err != nil {
				http.Error(w, "organization context required", http.StatusForbidden)
				return
			}
			if input.AssignedTo != nil {
				if _, err := users.GetByIDInOrganization(r.Context(), orgID, *input.AssignedTo); err != nil {
					if err == store.ErrUserNotFound {
						http.Error(w, "assigned user not found", http.StatusBadRequest)
						return
					}
					http.Error(w, "failed to validate assigned user", http.StatusInternalServerError)
					return
				}
			}
			incident, err := incidentService.Update(r.Context(), orgID, incidentID, claims.UserID, input)
			if err != nil {
				if errors.Is(err, store.ErrInvalidIncidentTransition) {
					http.Error(w, "invalid incident status transition", http.StatusConflict)
					return
				}
				if err == store.ErrIncidentNotFound {
					http.Error(w, "incident not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to update incident", http.StatusInternalServerError)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentUpdated, clientIP(r), map[string]interface{}{"incident_id": incident.ID.String(), "reference": incident.Reference})
			if err := cache.Delete(r.Context(), incidentsCacheKeyDefault(orgID)); err != nil {
				log.Printf("incidents: cache invalidation failed after update: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(incident)
		})

		r.With(a.perm("incidents.manage")).Post("/{incidentID}/notes", func(w http.ResponseWriter, r *http.Request) {
			incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
			if err != nil {
				http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
				return
			}
			var body struct {
				Message string `json:"message"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Message = strings.TrimSpace(body.Message)
			if err := validate.IncidentNote(body.Message); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if err := func() error {
				orgID, ok := requestOrganizationID(w, r)
				if !ok {
					return store.ErrIncidentNotFound
				}
				return incidentService.AddNote(r.Context(), orgID, incidentID, claims.UserID, body.Message)
			}(); err != nil {
				if err == store.ErrIncidentNotFound {
					http.Error(w, "incident not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to add note", http.StatusInternalServerError)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentNoteAdded, clientIP(r), map[string]interface{}{"incident_id": incidentID.String()})
			w.WriteHeader(http.StatusNoContent)
		})

		r.With(a.perm("incidents.manage")).Post("/{incidentID}/evidence", func(w http.ResponseWriter, r *http.Request) {
			incidentID, err := uuid.Parse(chi.URLParam(r, "incidentID"))
			if err != nil {
				http.Error(w, "incidentID must be a valid UUID", http.StatusBadRequest)
				return
			}
			var body struct {
				EvidenceType string `json:"evidence_type"`
				EvidenceRef  string `json:"evidence_ref"`
				Label        string `json:"label"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.EvidenceType = strings.ToLower(strings.TrimSpace(body.EvidenceType))
			body.EvidenceRef = strings.TrimSpace(body.EvidenceRef)
			body.Label = strings.TrimSpace(body.Label)
			if err := validate.IncidentEvidenceType(body.EvidenceType); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.IncidentEvidenceRef(body.EvidenceRef); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.IncidentEvidenceLabel(body.Label); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			evidence, err := func() (store.IncidentEvidence, error) {
				orgID, ok := requestOrganizationID(w, r)
				if !ok {
					return store.IncidentEvidence{}, store.ErrIncidentNotFound
				}
				return incidentService.AddEvidence(r.Context(), orgID, incidentID, claims.UserID, store.CreateIncidentEvidenceInput{
					EvidenceType: body.EvidenceType, EvidenceRef: body.EvidenceRef, Label: body.Label,
				})
			}()
			if err != nil {
				if errors.Is(err, store.ErrIncidentNotFound) {
					http.Error(w, "incident not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to add incident evidence", http.StatusInternalServerError)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionIncidentEvidenceAdded, clientIP(r), map[string]interface{}{"incident_id": incidentID.String(), "evidence_type": body.EvidenceType})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(evidence)
		})
	})

	r.Route("/asset-alerts", func(r chi.Router) {
		r.With(a.perm("alerts.read")).Get("/", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			cacheKey := assetAlertsCacheKey(orgID, limit, offset)
			var page store.Page[store.AssetCVEAlert]
			if hit, _ := cache.Get(r.Context(), cacheKey, &page); hit {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(page)
				return
			}
			page, err := assetAlerts.ListByOrganization(r.Context(), orgID, limit, offset)
			if err != nil {
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
				return
			}
			cache.Set(r.Context(), cacheKey, page, 3*time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.With(a.perm("alerts.manage")).Post("/{alertID}/acknowledge", func(w http.ResponseWriter, r *http.Request) {
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
			orgID, err := auth.OrganizationID(claims)
			if err != nil {
				http.Error(w, "organization context required", http.StatusForbidden)
				return
			}
			found, err := assetAlerts.AcknowledgeInOrganization(r.Context(), orgID, alertID, claims.UserID)
			if err != nil {
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Unexpected error", err)
				return
			}
			if !found {
				http.Error(w, "alert not found", http.StatusNotFound)
				return
			}

			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAssetAlertAcked, clientIP(r), map[string]interface{}{"alert_id": alertID.String()})
			if err := cache.Delete(r.Context(), assetAlertsCacheKeyDefault(orgID)); err != nil {
				log.Printf("asset-alerts: cache invalidation failed after acknowledge: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
}
