package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"net/http"
	"strings"
	"time"
)

func (a *apiRoutes) registerAgentKeyDeviceRoutes(r chi.Router) {
	agentIdentities := a.agentIdentities
	agentKeys := a.agentKeys
	auditLog := a.auditLog
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	rbac := a.rbac
	tenantLimiter := a.tenantLimiter

	r.Route("/agent-keys", func(r chi.Router) {
		agentRead := auth.RequireAnyPermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, []string{"agents.read", "agents.manage"}, isTokenRevoked)
		agentManage := auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "agents.manage", isTokenRevoked)

		r.With(agentRead).Get("/", func(w http.ResponseWriter, r *http.Request) {
			list, err := func() ([]store.AgentKeySummary, error) {
				orgID, ok := requestOrganizationID(w, r)
				if !ok {
					return nil, store.ErrAgentKeyNotFound
				}
				return agentKeys.ListByOrganization(r.Context(), orgID)
			}()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})

		r.With(agentManage).Post("/", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				AgentID string `json:"agent_id"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.AgentID = strings.TrimSpace(body.AgentID)
			if err := validate.AgentID(body.AgentID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			plaintext, hash, err := auth.GenerateAgentKey()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := func() error {
				orgID, ok := requestOrganizationID(w, r)
				if !ok {
					return store.ErrAgentKeyNotFound
				}
				return agentKeys.CreateInOrganization(r.Context(), orgID, body.AgentID, hash)
			}(); err != nil {
				if err == store.ErrAgentIDTaken {
					http.Error(w, "agent_id already has an active key; revoke it first", http.StatusConflict)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if claims, ok := auth.FromContext(r.Context()); ok {
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentKeyCreated, clientIP(r), map[string]interface{}{"agent_id": body.AgentID})
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"agent_id": body.AgentID,
				"api_key":  plaintext,
			})
		})

		r.With(agentManage).Delete("/{agentID}", func(w http.ResponseWriter, r *http.Request) {
			agentID := chi.URLParam(r, "agentID")
			if _, err := func() (time.Time, error) {
				orgID, ok := requestOrganizationID(w, r)
				if !ok {
					return time.Time{}, store.ErrAgentKeyNotFound
				}
				return agentKeys.RevokeInOrganization(r.Context(), orgID, agentID)
			}(); err != nil {
				if err == store.ErrAgentKeyNotFound {
					http.Error(w, "no active key for that agent_id", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if claims, ok := auth.FromContext(r.Context()); ok {
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentKeyRevoked, clientIP(r), map[string]interface{}{"agent_id": agentID})
			}

			w.WriteHeader(http.StatusNoContent)
		})
	})

	auditedAgentLookup := func(ctx context.Context, keyHash string) (string, error) {
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
	meAgentAuth := auth.RequireAgentCredential(auditedAgentLookup, auditedCertificateLookup)

	r.With(meAgentAuth).Post("/agents/me/device", func(w http.ResponseWriter, r *http.Request) {
		agentID, ok := auth.AgentIDFromContext(r.Context())
		if !ok {
			http.Error(w, "this endpoint requires agent key authentication", http.StatusUnauthorized)
			return
		}

		orgID, err := agentIdentities.OrganizationForAgent(r.Context(), agentID)
		if err != nil {
			orgID, err = agentKeys.OrganizationForAgent(r.Context(), agentID)
		}
		if err != nil {
			http.Error(w, "agent organization unavailable", http.StatusUnauthorized)
			return
		}
		if !enforceAgentRateLimit(w, r, tenantLimiter, orgID, agentID) {
			return
		}

		var body store.AgentDeviceInfo
		if err := decodeStrictJSON(r, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := validateAgentDeviceReport(body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := agentIdentities.ReportDeviceInOrganization(r.Context(), orgID, agentID, body); err != nil {
			if errors.Is(err, store.ErrAgentIdentityNotFound) {
				if err := agentKeys.ReportDeviceInOrganization(r.Context(), orgID, agentID, body); err != nil {
					http.Error(w, "unknown or inactive agent", http.StatusUnauthorized)
					return
				}
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorAgent, agentID, audit.ActionAgentHardwareReported, clientIP(r), map[string]interface{}{
			"driver": body.Driver, "hardware": body.Hardware, "serial": body.Serial,
		})
		w.WriteHeader(http.StatusNoContent)
	})
}
