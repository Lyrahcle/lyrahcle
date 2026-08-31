package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

type signingKeyListResponse struct {
	Provider           string                      `json:"provider"`
	Managed            bool                        `json:"managed"`
	GracePeriodSeconds int64                       `json:"grace_period_seconds"`
	Keys               []keyprovider.KeyMetadata   `json:"keys"`
	RotationPolicy     *keyprovider.RotationPolicy `json:"rotation_policy,omitempty"`
	Health             *keyprovider.RotationHealth `json:"health,omitempty"`
}

func registerSigningKeyRoutes(
	r chi.Router,
	cfg config.Config,
	provider keyprovider.Provider,
	users *store.UserStore,
	auditLog *store.AuditStore,
	isTokenRevoked func(context.Context, *auth.Claims) (bool, error),
) {
	requirePlatformAdmin := func(next http.Handler) http.Handler {
		return auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			allowed, err := users.IsPlatformAdministrator(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "platform authorization unavailable", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "platform administrator required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}

	r.Route("/signing-keys/agent-config", func(r chi.Router) {
		r.Use(requirePlatformAdmin)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			response := signingKeyListResponse{
				Provider:           "disabled",
				Managed:            false,
				GracePeriodSeconds: int64(cfg.AgentConfigSigningGracePeriod / time.Second),
				Keys:               []keyprovider.KeyMetadata{},
			}
			if provider == nil {
				writeSigningJSON(w, response)
				return
			}
			if lifecycle, ok := provider.(keyprovider.LifecycleProvider); ok {
				keys, err := lifecycle.Keys(r.Context())
				if err != nil {
					http.Error(w, "failed to list signing keys", http.StatusInternalServerError)
					return
				}
				policy, err := lifecycle.RotationPolicy(r.Context())
				if err != nil {
					http.Error(w, "failed to read signing key rotation policy", http.StatusInternalServerError)
					return
				}
				health := keyprovider.AssessRotation(keys, policy, time.Now().UTC())
				metrics.RecordSigningKeyLifecycle(agentConfigSigningDomain, keys, health)
				response.Provider = "local-managed"
				response.Managed = true
				response.Keys = keys
				response.RotationPolicy = &policy
				response.Health = &health
				writeSigningJSON(w, response)
				return
			}
			active, err := provider.ActiveSigningKey(r.Context())
			if err != nil {
				http.Error(w, "failed to read active signing key", http.StatusInternalServerError)
				return
			}
			response.Provider = "local-legacy"
			response.Keys = []keyprovider.KeyMetadata{{KID: active.KID(), Algorithm: active.Algorithm(), Status: keyprovider.KeyStatusActive}}
			writeSigningJSON(w, response)
		})

		r.Put("/policy", func(w http.ResponseWriter, r *http.Request) {
			lifecycle, ok := provider.(keyprovider.LifecycleProvider)
			if !ok {
				http.Error(w, "managed signing key lifecycle is not configured", http.StatusConflict)
				return
			}
			var body keyprovider.RotationPolicy
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			updated, err := lifecycle.UpdateRotationPolicy(r.Context(), body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			logSigningKeyAudit(r, auditLog, audit.ActionSigningKeyPolicyUpdated, "", map[string]interface{}{
				"automatic_prepare":          updated.AutomaticPrepare,
				"interval_seconds":           updated.IntervalSeconds,
				"staged_alert_after_seconds": updated.StagedAlertAfterSeconds,
				"grace_alert_before_seconds": updated.GraceAlertBeforeSeconds,
			})
			writeSigningJSON(w, updated)
		})

		r.Post("/prepare", func(w http.ResponseWriter, r *http.Request) {
			lifecycle, ok := provider.(keyprovider.LifecycleProvider)
			if !ok {
				http.Error(w, "managed signing key lifecycle is not configured", http.StatusConflict)
				return
			}
			metadata, publicPEM, err := lifecycle.PrepareRotation(r.Context())
			if err != nil {
				if errors.Is(err, keyprovider.ErrRotationPending) {
					http.Error(w, "a staged signing key already exists", http.StatusConflict)
					return
				}
				http.Error(w, "failed to prepare signing key rotation", http.StatusInternalServerError)
				return
			}
			logSigningKeyAudit(r, auditLog, audit.ActionSigningKeyPrepared, metadata.KID, map[string]interface{}{
				"status": metadata.Status, "algorithm": metadata.Algorithm,
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"key": metadata, "public_key_pem": string(publicPEM)})
		})

		r.Get("/{kid}/public-key", func(w http.ResponseWriter, r *http.Request) {
			lifecycle, ok := provider.(keyprovider.LifecycleProvider)
			if !ok {
				http.Error(w, "managed signing key lifecycle is not configured", http.StatusConflict)
				return
			}
			kid := strings.TrimSpace(chi.URLParam(r, "kid"))
			if err := keyprovider.ValidateKID(kid); err != nil {
				http.Error(w, "invalid signing key id", http.StatusBadRequest)
				return
			}
			publicPEM, err := lifecycle.PublicKeyPEM(r.Context(), kid)
			if err != nil {
				if errors.Is(err, keyprovider.ErrKeyNotFound) {
					http.Error(w, "signing key not found", http.StatusNotFound)
					return
				}
				http.Error(w, "failed to read signing public key", http.StatusInternalServerError)
				return
			}
			writeSigningJSON(w, map[string]string{"kid": kid, "public_key_pem": string(publicPEM)})
		})

		r.Post("/{kid}/activate", func(w http.ResponseWriter, r *http.Request) {
			lifecycle, ok := provider.(keyprovider.LifecycleProvider)
			if !ok {
				http.Error(w, "managed signing key lifecycle is not configured", http.StatusConflict)
				return
			}
			kid := strings.TrimSpace(chi.URLParam(r, "kid"))
			if err := keyprovider.ValidateKID(kid); err != nil {
				http.Error(w, "invalid signing key id", http.StatusBadRequest)
				return
			}
			var body struct {
				GraceSeconds int64 `json:"grace_seconds"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			grace := cfg.AgentConfigSigningGracePeriod
			if body.GraceSeconds != 0 {
				grace = time.Duration(body.GraceSeconds) * time.Second
			}
			if grace < time.Hour || grace > 30*24*time.Hour {
				http.Error(w, "grace_seconds must be between 3600 and 2592000", http.StatusBadRequest)
				return
			}
			metadata, err := lifecycle.Activate(r.Context(), kid, grace)
			if err != nil {
				switch {
				case errors.Is(err, keyprovider.ErrKeyNotFound):
					http.Error(w, "signing key not found", http.StatusNotFound)
				case errors.Is(err, keyprovider.ErrKeyNotStaged):
					http.Error(w, "only a staged signing key can be activated", http.StatusConflict)
				default:
					http.Error(w, "failed to activate signing key", http.StatusInternalServerError)
				}
				return
			}
			logSigningKeyAudit(r, auditLog, audit.ActionSigningKeyActivated, metadata.KID, map[string]interface{}{
				"status": metadata.Status, "grace_seconds": int64(grace / time.Second),
			})
			writeSigningJSON(w, metadata)
		})

		r.Post("/{kid}/revoke", func(w http.ResponseWriter, r *http.Request) {
			lifecycle, ok := provider.(keyprovider.LifecycleProvider)
			if !ok {
				http.Error(w, "managed signing key lifecycle is not configured", http.StatusConflict)
				return
			}
			kid := strings.TrimSpace(chi.URLParam(r, "kid"))
			if err := keyprovider.ValidateKID(kid); err != nil {
				http.Error(w, "invalid signing key id", http.StatusBadRequest)
				return
			}
			metadata, err := lifecycle.Revoke(r.Context(), kid)
			if err != nil {
				switch {
				case errors.Is(err, keyprovider.ErrKeyNotFound):
					http.Error(w, "signing key not found", http.StatusNotFound)
				case errors.Is(err, keyprovider.ErrActiveKey):
					http.Error(w, "active signing key cannot be revoked; activate a replacement first", http.StatusConflict)
				default:
					http.Error(w, "failed to revoke signing key", http.StatusInternalServerError)
				}
				return
			}
			logSigningKeyAudit(r, auditLog, audit.ActionSigningKeyRevoked, metadata.KID, map[string]interface{}{"status": metadata.Status})
			writeSigningJSON(w, metadata)
		})
	})
}

func writeSigningJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func logSigningKeyAudit(r *http.Request, auditLog *store.AuditStore, action, kid string, details map[string]interface{}) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return
	}
	if details == nil {
		details = map[string]interface{}{}
	}
	if kid != "" {
		details["kid"] = kid
	}
	details["domain"] = agentConfigSigningDomain
	logAuditInOrganization(r.Context(), auditLog, claims.OrganizationID, audit.ActorUser, claims.UserID.String(), action, clientIP(r), details)
}
