package main

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/dbscope"
	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

func (a *apiRoutes) registerDetectionRFRoutes(r chi.Router) {
	agentIdentities := a.agentIdentities
	agentKeys := a.agentKeys
	assetAlerts := a.assetAlerts
	auditLog := a.auditLog
	cache := a.cache
	cfg := a.cfg
	detectionRules := a.detectionRules
	incidents := a.incidents
	isTokenRevoked := a.isTokenRevoked
	orbitalEvents := a.orbitalEvents
	rbac := a.rbac
	rf := a.rf
	rfAnomalies := a.rfAnomalies
	satellites := a.satellites
	spaceCVEs := a.spaceCVEs
	tenantLimiter := a.tenantLimiter

	r.Route("/detection-rules", func(r chi.Router) {
		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "detections.read", isTokenRevoked)).Get("/", func(w http.ResponseWriter, r *http.Request) {
			limit, offset := pageParams(r)
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			page, err := detectionRules.ListByOrganization(r.Context(), orgID, limit, offset)
			if err != nil {
				http.Error(w, "failed to fetch detection rules", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "detections.manage", isTokenRevoked)).Post("/", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Name            string  `json:"name"`
				Description     string  `json:"description"`
				Enabled         *bool   `json:"enabled"`
				Metric          string  `json:"metric"`
				Operator        string  `json:"operator"`
				Threshold       float64 `json:"threshold"`
				Severity        string  `json:"severity"`
				CooldownSeconds int     `json:"cooldown_seconds"`
				AutoIncident    *bool   `json:"auto_incident"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Name = strings.TrimSpace(body.Name)
			body.Description = strings.TrimSpace(body.Description)
			body.Metric = strings.ToLower(strings.TrimSpace(body.Metric))
			body.Operator = strings.ToLower(strings.TrimSpace(body.Operator))
			body.Severity = strings.ToLower(strings.TrimSpace(body.Severity))
			if err := validateDetectionRule(body.Name, body.Description, body.Metric, body.Operator, body.Threshold, body.Severity, body.CooldownSeconds); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
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
			enabled := true
			if body.Enabled != nil {
				enabled = *body.Enabled
			}
			if enabled {
				count, err := detectionRules.EnabledCountByOrganization(r.Context(), orgID)
				if err != nil {
					http.Error(w, "failed to validate detection rule capacity", http.StatusInternalServerError)
					return
				}
				if count >= store.MaxEnabledDetectionRules {
					http.Error(w, "maximum enabled detection rules reached", http.StatusConflict)
					return
				}
			}
			autoIncident := true
			if body.AutoIncident != nil {
				autoIncident = *body.AutoIncident
			}
			rule, err := detectionRules.CreateInOrganization(r.Context(), orgID, detection.Rule{
				Name: body.Name, Description: body.Description, Enabled: enabled, Metric: body.Metric,
				Operator: body.Operator, Threshold: body.Threshold, Severity: body.Severity,
				CooldownSeconds: body.CooldownSeconds, AutoIncident: autoIncident,
			}, claims.UserID)
			if err != nil {
				http.Error(w, "failed to create detection rule", http.StatusConflict)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionDetectionRuleCreated, clientIP(r), map[string]interface{}{"organization_id": orgID.String(), "rule_id": rule.ID.String(), "name": rule.Name})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(rule)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "detections.manage", isTokenRevoked)).Patch("/{ruleID}", func(w http.ResponseWriter, r *http.Request) {
			ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
			if err != nil {
				http.Error(w, "ruleID must be a valid UUID", http.StatusBadRequest)
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			rule, err := detectionRules.GetByOrganization(r.Context(), orgID, ruleID)
			if err != nil {
				if err == store.ErrDetectionRuleNotFound {
					http.Error(w, "detection rule not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to fetch detection rule", http.StatusInternalServerError)
				return
			}
			wasEnabled := rule.Enabled
			var body struct {
				Name            *string  `json:"name"`
				Description     *string  `json:"description"`
				Enabled         *bool    `json:"enabled"`
				Metric          *string  `json:"metric"`
				Operator        *string  `json:"operator"`
				Threshold       *float64 `json:"threshold"`
				Severity        *string  `json:"severity"`
				CooldownSeconds *int     `json:"cooldown_seconds"`
				AutoIncident    *bool    `json:"auto_incident"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if body.Name != nil {
				rule.Name = strings.TrimSpace(*body.Name)
			}
			if body.Description != nil {
				rule.Description = strings.TrimSpace(*body.Description)
			}
			if body.Enabled != nil {
				rule.Enabled = *body.Enabled
			}
			if body.Metric != nil {
				rule.Metric = strings.ToLower(strings.TrimSpace(*body.Metric))
			}
			if body.Operator != nil {
				rule.Operator = strings.ToLower(strings.TrimSpace(*body.Operator))
			}
			if body.Threshold != nil {
				rule.Threshold = *body.Threshold
			}
			if body.Severity != nil {
				rule.Severity = strings.ToLower(strings.TrimSpace(*body.Severity))
			}
			if body.CooldownSeconds != nil {
				rule.CooldownSeconds = *body.CooldownSeconds
			}
			if body.AutoIncident != nil {
				rule.AutoIncident = *body.AutoIncident
			}
			if err := validateDetectionRule(rule.Name, rule.Description, rule.Metric, rule.Operator, rule.Threshold, rule.Severity, rule.CooldownSeconds); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if rule.Enabled && !wasEnabled {
				count, err := detectionRules.EnabledCountByOrganization(r.Context(), orgID)
				if err != nil {
					http.Error(w, "failed to validate detection rule capacity", http.StatusInternalServerError)
					return
				}
				if count >= store.MaxEnabledDetectionRules {
					http.Error(w, "maximum enabled detection rules reached", http.StatusConflict)
					return
				}
			}
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			rule, err = detectionRules.UpdateInOrganization(r.Context(), orgID, rule, claims.UserID)
			if err != nil {
				http.Error(w, "failed to update detection rule", http.StatusInternalServerError)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionDetectionRuleUpdated, clientIP(r), map[string]interface{}{"organization_id": orgID.String(), "rule_id": rule.ID.String(), "name": rule.Name, "enabled": rule.Enabled})
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rule)
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
		auditedCertificateLookup := func(ctx context.Context, fingerprint string) (string, error) {
			agentID, err := agentIdentities.AgentIDForCertificate(ctx, fingerprint)
			if err != nil {
				logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentCertificateAuthFailed, "", nil)
			}
			return agentID, err
		}
		agentAuth := auth.RequireAgentCredentialOrPermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "rf.ingest", auditedLookup, auditedCertificateLookup, isTokenRevoked)
		r.With(agentAuth).Post("/ingest", func(w http.ResponseWriter, r *http.Request) {
			if authenticatedAgentID, ok := auth.AgentIDFromContext(r.Context()); ok {
				orgID, err := agentIdentities.OrganizationForAgent(r.Context(), authenticatedAgentID)
				if err != nil {
					orgID, err = agentKeys.OrganizationForAgent(r.Context(), authenticatedAgentID)
				}
				if err != nil {
					http.Error(w, "agent organization unavailable", http.StatusUnauthorized)
					return
				}
				if !enforceAgentRateLimit(w, r, tenantLimiter, orgID, authenticatedAgentID) {
					return
				}
			}
			var reading store.RFReading
			if err := decodeStrictJSON(r, &reading); err != nil {
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

			if reading.SatelliteID != nil {
				exists, err := satellites.Exists(r.Context(), *reading.SatelliteID)
				if err != nil {
					http.Error(w, "failed to validate satellite", http.StatusInternalServerError)
					return
				}
				if !exists {
					http.Error(w, "satellite not found", http.StatusBadRequest)
					return
				}
			}

			if authenticatedAgentID, ok := auth.AgentIDFromContext(r.Context()); ok {
				if reading.AgentID != authenticatedAgentID {
					http.Error(w, "agent_id does not match the authenticated agent key", http.StatusForbidden)
					return
				}
			}

			var organizationID uuid.UUID
			if claims, ok := auth.FromContext(r.Context()); ok && claims.OrganizationID != uuid.Nil {
				organizationID = claims.OrganizationID
			} else if id, err := agentIdentities.OrganizationForAgent(r.Context(), reading.AgentID); err == nil {
				organizationID = id
			} else if id, err := agentKeys.OrganizationForAgent(r.Context(), reading.AgentID); err == nil {
				organizationID = id
			} else {
				http.Error(w, "agent organization unavailable", http.StatusForbidden)
				return
			}
			reading.OrganizationID = organizationID
			r = r.WithContext(dbscope.WithOrganization(r.Context(), organizationID))

			history, err := rf.RecentForAgentInOrganization(r.Context(), organizationID, reading.AgentID, anomalyHistoryWindow)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := rf.Insert(r.Context(), &reading); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			detectRFAnomalies(r.Context(), rfAnomalies, detectionRules, incidents, auditLog, assetAlerts, satellites, agentIdentities, spaceCVEs, orbitalEvents, tlePolicy(cfg), cfg.OrbitalAutoEscalation, reading, history)

			if err := cache.Delete(r.Context(), rfReadingsCacheKey(organizationID)); err != nil {
				log.Printf("rf: cache invalidation failed after ingest (readings): %v", err)
			}
			if err := cache.Delete(r.Context(), rfAnomaliesCacheKey(organizationID)); err != nil {
				log.Printf("rf: cache invalidation failed after ingest (anomalies): %v", err)
			}

			w.WriteHeader(http.StatusCreated)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "rf.read", isTokenRevoked)).Get("/readings", func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			cacheKey := rfReadingsCacheKey(orgID)
			var list []store.RFReading
			if hit, _ := cache.Get(r.Context(), cacheKey, &list); hit {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
				return
			}
			list, err := rf.RecentByOrganization(r.Context(), orgID, 100)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cache.Set(r.Context(), cacheKey, list, 3*time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})

		r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "rf.read", isTokenRevoked)).Get("/anomalies", func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			cacheKey := rfAnomaliesCacheKey(orgID)
			var list []store.RFAnomaly
			if hit, _ := cache.Get(r.Context(), cacheKey, &list); hit {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(list)
				return
			}
			list, err := rfAnomalies.RecentByOrganization(r.Context(), orgID, 100)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			cache.Set(r.Context(), cacheKey, list, 3*time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})
	})
}
