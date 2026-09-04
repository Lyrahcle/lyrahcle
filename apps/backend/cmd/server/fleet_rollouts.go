package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/agentconfig"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func registerFleetRolloutRoutes(r chi.Router, cfg config.Config, signer *agentconfig.Signer, rollouts *store.FleetRolloutStore, configs *store.AgentConfigStore, satellites *store.SatelliteStore, auditLog *store.AuditStore, rbac *store.RBACStore, isTokenRevoked auth.RevocationCheck) {
	read := auth.RequireAnyPermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, []string{"agents.read", "agents.manage"}, isTokenRevoked)
	manage := newPermissionMiddleware(cfg, rbac, isTokenRevoked)("agents.manage")

	r.With(read).Get("/agent-rollouts", func(w http.ResponseWriter, req *http.Request) {
		orgID, ok := requestOrganizationID(w, req)
		if !ok {
			return
		}
		items, err := rollouts.List(req.Context(), orgID)
		if err != nil {
			http.Error(w, "failed to list rollouts", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	r.With(read).Get("/agent-rollouts/{rolloutID}", func(w http.ResponseWriter, req *http.Request) {
		orgID, ok := requestOrganizationID(w, req)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(req, "rolloutID"))
		if err != nil {
			http.Error(w, "invalid rollout id", 400)
			return
		}
		item, err := rollouts.Get(req.Context(), orgID, id)
		if errors.Is(err, store.ErrFleetRolloutNotFound) {
			http.Error(w, "rollout not found", 404)
			return
		}
		if err != nil {
			http.Error(w, "failed to read rollout", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	})

	r.With(manage).Post("/agent-rollouts", func(w http.ResponseWriter, req *http.Request) {
		if signer == nil {
			http.Error(w, "agent remote configuration is not configured", 503)
			return
		}
		var body struct {
			Name                    string                    `json:"name"`
			AgentIDs                []string                  `json:"agent_ids"`
			Config                  store.AgentConfigSettings `json:"config"`
			BatchSize               int                       `json:"batch_size"`
			FailureThresholdPercent int                       `json:"failure_threshold_percent"`
			SoakSeconds             int                       `json:"soak_seconds"`
			AckTimeoutSeconds       int                       `json:"ack_timeout_seconds"`
		}
		if err := decodeAgentJSON(w, req, 16<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", 400)
			return
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" || len(body.Name) > 120 || len(body.AgentIDs) == 0 {
			http.Error(w, "name and agent_ids are required", 400)
			return
		}
		if err := validateRemoteSettings(body.Config); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if body.BatchSize <= 0 {
			body.BatchSize = 1
		}
		if body.BatchSize > len(body.AgentIDs) {
			body.BatchSize = len(body.AgentIDs)
		}
		if body.FailureThresholdPercent < 0 || body.FailureThresholdPercent > 100 {
			http.Error(w, "failure_threshold_percent must be 0-100", 400)
			return
		}
		if body.SoakSeconds < 0 || body.SoakSeconds > 86400 {
			http.Error(w, "soak_seconds must be 0-86400", 400)
			return
		}
		if body.AckTimeoutSeconds == 0 {
			body.AckTimeoutSeconds = 900
		}
		if body.AckTimeoutSeconds < 30 || body.AckTimeoutSeconds > 86400 {
			http.Error(w, "ack_timeout_seconds must be 30-86400", 400)
			return
		}
		if body.Config.SatelliteID != nil {
			exists, err := satellites.Exists(req.Context(), *body.Config.SatelliteID)
			if err != nil || !exists {
				http.Error(w, "satellite not found", 400)
				return
			}
		}
		seen := map[string]bool{}
		for i, id := range body.AgentIDs {
			id = strings.TrimSpace(id)
			body.AgentIDs[i] = id
			if id == "" || seen[id] {
				http.Error(w, "agent_ids must be unique and non-empty", 400)
				return
			}
			seen[id] = true
		}
		claims, _ := auth.FromContext(req.Context())
		orgID, err := auth.OrganizationID(claims)
		if err != nil {
			http.Error(w, "organization context required", 403)
			return
		}
		eligibility, err := rollouts.EvaluateTargets(req.Context(), orgID, body.AgentIDs, body.Config.FrequencyHz)
		if err != nil {
			http.Error(w, "failed to validate rollout target policy", 500)
			return
		}
		if reason := fleetEligibilityFailure(eligibility); reason != "" {
			http.Error(w, "rollout target policy rejected: "+reason, http.StatusUnprocessableEntity)
			return
		}
		createdBy := claims.UserID
		item, err := rollouts.Create(req.Context(), orgID, body.Name, body.Config, body.BatchSize, body.FailureThresholdPercent, body.SoakSeconds, body.AckTimeoutSeconds, body.AgentIDs, &createdBy)
		if errors.Is(err, store.ErrAgentIdentityNotFound) {
			http.Error(w, "all rollout agents must be active and belong to the organization", 400)
			return
		}
		if err != nil {
			http.Error(w, "failed to create rollout", 500)
			return
		}
		logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentRolloutCreated, clientIP(req), map[string]interface{}{"rollout_id": item.ID, "targets": len(item.Targets), "batch_size": item.BatchSize, "ack_timeout_seconds": item.AckTimeoutSeconds})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(item)
	})

	issueBatch := func(w http.ResponseWriter, req *http.Request, start bool) {
		if signer == nil {
			http.Error(w, "agent remote configuration is not configured", 503)
			return
		}
		orgID, ok := requestOrganizationID(w, req)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(req, "rolloutID"))
		if err != nil {
			http.Error(w, "invalid rollout id", 400)
			return
		}
		item, err := rollouts.Get(req.Context(), orgID, id)
		if err != nil {
			http.Error(w, "rollout not found", 404)
			return
		}
		claims, _ := auth.FromContext(req.Context())
		if err := rollouts.RefreshAcknowledgements(req.Context(), orgID, id); err != nil {
			http.Error(w, "failed to refresh rollout", 500)
			return
		}
		pending, issued, applied, rejected, err := rollouts.Counts(req.Context(), orgID, id)
		if err != nil {
			http.Error(w, "failed to inspect rollout", 500)
			return
		}
		decided := applied + rejected
		if decided > 0 && rejected*100/decided > item.FailureThresholdPercent {
			_ = rollouts.SetStatus(req.Context(), orgID, id, []string{"running", "paused"}, "failed")
			logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentRolloutFailed, clientIP(req), map[string]interface{}{"rollout_id": id, "reason": "failure_threshold", "applied": applied, "rejected": rejected})
			http.Error(w, "rollout failure threshold exceeded", 409)
			return
		}
		if issued > 0 {
			overdue, err := rollouts.OverdueIssuedCount(req.Context(), orgID, id, item.AckTimeoutSeconds)
			if err != nil {
				http.Error(w, "failed to inspect rollout acknowledgement deadline", 500)
				return
			}
			if overdue > 0 {
				_ = rollouts.SetStatus(req.Context(), orgID, id, []string{"running", "paused"}, "failed")
				logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentRolloutFailed, clientIP(req), map[string]interface{}{"rollout_id": id, "reason": "ack_timeout", "overdue_targets": overdue, "ack_timeout_seconds": item.AckTimeoutSeconds})
				http.Error(w, "rollout acknowledgement timeout exceeded", 409)
				return
			}
			http.Error(w, "wait for the current batch acknowledgements before advancing", 409)
			return
		}
		if !start && decided > 0 && item.SoakSeconds > 0 {
			latestIssuedAt, err := rollouts.LatestIssuedAt(req.Context(), orgID, id)
			if err != nil {
				http.Error(w, "failed to inspect rollout soak period", 500)
				return
			}
			if latestIssuedAt != nil && time.Since(*latestIssuedAt) < time.Duration(item.SoakSeconds)*time.Second {
				http.Error(w, "rollout soak period has not elapsed", 409)
				return
			}
		}
		allowed := []string{"running"}
		if start {
			allowed = []string{"draft"}
		}
		targets, err := rollouts.NextPending(req.Context(), orgID, id, item.BatchSize)
		if err != nil {
			http.Error(w, "failed to select rollout batch", 500)
			return
		}
		targetIDs := make([]string, 0, len(targets))
		for _, target := range targets {
			targetIDs = append(targetIDs, target.AgentID)
		}
		eligibility, err := rollouts.EvaluateTargets(req.Context(), orgID, targetIDs, item.Settings.FrequencyHz)
		if err != nil {
			http.Error(w, "failed to revalidate rollout target policy", 500)
			return
		}
		if reason := fleetEligibilityFailure(eligibility); reason != "" {
			_ = rollouts.SetStatus(req.Context(), orgID, id, allowed, "failed")
			logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentRolloutFailed, clientIP(req), map[string]interface{}{"rollout_id": id, "reason": "target_policy", "details": reason})
			http.Error(w, "rollout target policy changed: "+reason, 409)
			return
		}
		if err := rollouts.SetStatus(req.Context(), orgID, id, allowed, "running"); err != nil {
			http.Error(w, "rollout is not in a promotable state", 409)
			return
		}
		createdBy := claims.UserID
		for _, target := range targets {
			var previous *int64
			if current, e := configs.DesiredInOrganization(req.Context(), orgID, target.AgentID); e == nil {
				v := current.Version
				previous = &v
			}
			created, err := configs.CreateInOrganization(req.Context(), orgID, target.AgentID, item.Settings, &createdBy, func(version int64, value store.AgentConfigSettings, issuedAt time.Time) (store.AgentConfigSignResult, error) {
				envelope, payload, signature, err := signer.Sign(req.Context(), target.AgentID, version, agentconfig.Settings{FrequencyHz: value.FrequencyHz, CaptureIntervalSeconds: value.CaptureIntervalSeconds, SatelliteID: value.SatelliteID}, issuedAt)
				if err != nil {
					return store.AgentConfigSignResult{}, err
				}
				return store.AgentConfigSignResult{Payload: payload, Signature: signature, Checksum: envelope.Checksum, SigningKID: envelope.KID}, nil
			})
			if err != nil {
				_ = rollouts.SetStatus(req.Context(), orgID, id, []string{"running"}, "failed")
				logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentRolloutFailed, clientIP(req), map[string]interface{}{"rollout_id": id, "reason": "config_issue", "agent_id": target.AgentID})
				http.Error(w, "failed to issue rollout batch", 500)
				return
			}
			if err := rollouts.MarkIssued(req.Context(), orgID, id, target.AgentID, previous, created.Version); err != nil {
				http.Error(w, "failed to persist rollout progress", 500)
				return
			}
		}
		pending, issued, applied, rejected, _ = rollouts.Counts(req.Context(), orgID, id)
		if pending == 0 && issued == 0 {
			_ = rollouts.SetStatus(req.Context(), orgID, id, []string{"running"}, "completed")
		}
		updated, _ := rollouts.Get(req.Context(), orgID, id)
		action := audit.ActionAgentRolloutAdvanced
		if start {
			action = audit.ActionAgentRolloutStarted
		}
		logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), action, clientIP(req), map[string]interface{}{"rollout_id": id, "batch": len(targets)})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)
	}
	r.With(manage).Post("/agent-rollouts/{rolloutID}/start", func(w http.ResponseWriter, req *http.Request) { issueBatch(w, req, true) })
	r.With(manage).Post("/agent-rollouts/{rolloutID}/advance", func(w http.ResponseWriter, req *http.Request) { issueBatch(w, req, false) })
	r.With(manage).Post("/agent-rollouts/{rolloutID}/pause", func(w http.ResponseWriter, req *http.Request) {
		changeRolloutStatus(w, req, rollouts, auditLog, []string{"running"}, "paused", audit.ActionAgentRolloutPaused)
	})
	r.With(manage).Post("/agent-rollouts/{rolloutID}/resume", func(w http.ResponseWriter, req *http.Request) {
		changeRolloutStatus(w, req, rollouts, auditLog, []string{"paused"}, "running", audit.ActionAgentRolloutResumed)
	})
	r.With(manage).Post("/agent-rollouts/{rolloutID}/rollback", func(w http.ResponseWriter, req *http.Request) {
		orgID, ok := requestOrganizationID(w, req)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(req, "rolloutID"))
		if err != nil {
			http.Error(w, "invalid rollout id", 400)
			return
		}
		item, err := rollouts.Get(req.Context(), orgID, id)
		if err != nil {
			http.Error(w, "rollout not found", 404)
			return
		}
		claims, _ := auth.FromContext(req.Context())
		createdBy := claims.UserID
		for _, target := range item.Targets {
			if target.Status == "pending" || target.Status == "rolled_back" {
				continue
			}
			if target.PreviousVersion == nil {
				if err := configs.ClearDesiredInOrganization(req.Context(), orgID, target.AgentID); err == nil || errors.Is(err, store.ErrAgentConfigNotFound) {
					_ = rollouts.MarkRolledBack(req.Context(), orgID, id, target.AgentID)
				}
				continue
			}
			settings, err := configs.SettingsForVersionInOrganization(req.Context(), orgID, target.AgentID, *target.PreviousVersion)
			if err != nil {
				continue
			}
			_, err = configs.CreateInOrganization(req.Context(), orgID, target.AgentID, settings, &createdBy, func(version int64, value store.AgentConfigSettings, issuedAt time.Time) (store.AgentConfigSignResult, error) {
				env, payload, sig, err := signer.Sign(req.Context(), target.AgentID, version, agentconfig.Settings{FrequencyHz: value.FrequencyHz, CaptureIntervalSeconds: value.CaptureIntervalSeconds, SatelliteID: value.SatelliteID}, issuedAt)
				if err != nil {
					return store.AgentConfigSignResult{}, err
				}
				return store.AgentConfigSignResult{Payload: payload, Signature: sig, Checksum: env.Checksum, SigningKID: env.KID}, nil
			})
			if err == nil {
				_ = rollouts.MarkRolledBack(req.Context(), orgID, id, target.AgentID)
			}
		}
		_ = rollouts.SetStatus(req.Context(), orgID, id, []string{"draft", "running", "paused", "completed", "failed"}, "rolled_back")
		updated, _ := rollouts.Get(req.Context(), orgID, id)
		logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentRolloutRolledBack, clientIP(req), map[string]interface{}{"rollout_id": id})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)
	})
}

func fleetEligibilityFailure(items []store.FleetTargetEligibility) string {
	parts := make([]string, 0, 3)
	remaining := 0
	for _, item := range items {
		if item.Eligible {
			continue
		}
		if len(parts) < 3 {
			parts = append(parts, item.AgentID+": "+item.Reason)
		} else {
			remaining++
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if remaining > 0 {
		parts = append(parts, "and additional ineligible targets")
	}
	return strings.Join(parts, "; ")
}

func changeRolloutStatus(w http.ResponseWriter, req *http.Request, rollouts *store.FleetRolloutStore, auditLog *store.AuditStore, allowed []string, status, action string) {
	orgID, ok := requestOrganizationID(w, req)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(req, "rolloutID"))
	if err != nil {
		http.Error(w, "invalid rollout id", 400)
		return
	}
	if err := rollouts.SetStatus(req.Context(), orgID, id, allowed, status); err != nil {
		http.Error(w, "invalid rollout state transition", 409)
		return
	}
	item, _ := rollouts.Get(req.Context(), orgID, id)
	claims, _ := auth.FromContext(req.Context())
	logAudit(req.Context(), auditLog, audit.ActorUser, claims.UserID.String(), action, clientIP(req), map[string]interface{}{"rollout_id": id, "status": status})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}
