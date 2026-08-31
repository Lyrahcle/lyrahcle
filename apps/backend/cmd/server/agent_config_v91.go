package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mlko01/lyrahcle/apps/backend/internal/agentconfig"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

func buildAgentConfigSigner(cfg config.Config) (*agentconfig.Signer, keyprovider.Provider, error) {
	if cfg.AgentConfigSigningKeyDir == "" && cfg.AgentConfigSigningKeyFile == "" && cfg.AgentConfigSignerEndpoint == "" {
		return nil, nil, nil
	}
	var provider keyprovider.Provider
	var err error
	if cfg.AgentConfigSignerEndpoint != "" {
		provider, err = keyprovider.NewRemoteProvider(cfg.AgentConfigSignerEndpoint, cfg.AgentConfigSignerToken, cfg.AgentConfigSignerKID, cfg.AgentConfigSignerAlgorithm, 5*time.Second)
	} else if cfg.AgentConfigSigningKeyDir != "" {
		provider, err = keyprovider.NewManagedLocalProvider(cfg.AgentConfigSigningKeyDir, cfg.AgentConfigSigningKeyFile)
	} else {
		provider, err = keyprovider.NewLocalProvider(cfg.AgentConfigSigningKeyFile)
	}
	if err != nil {
		return nil, nil, err
	}
	signer, err := agentconfig.NewSigner(provider)
	if err != nil {
		return nil, nil, err
	}
	return signer, provider, nil
}

func validateRemoteSettings(settings store.AgentConfigSettings) error {
	if settings.FrequencyHz < 100000 || settings.FrequencyHz > 6000000000 {
		return errors.New("frequency_hz must be between 100000 and 6000000000")
	}
	if settings.CaptureIntervalSeconds < 1 || settings.CaptureIntervalSeconds > 3600 {
		return errors.New("capture_interval_seconds must be between 1 and 3600")
	}
	return nil
}

func registerAgentConfigRoutes(
	r chi.Router,
	cfg config.Config,
	signer *agentconfig.Signer,
	signingKeys keyprovider.Provider,
	configs *store.AgentConfigStore,
	identities *store.AgentIdentityStore,
	satellites *store.SatelliteStore,
	auditLog *store.AuditStore,
	rbac *store.RBACStore,
	isTokenRevoked auth.RevocationCheck,
	tenantLimiter *ratelimit.Limiter,
) {
	agentRead := auth.RequireAnyPermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, []string{"agents.read", "agents.manage"}, isTokenRevoked)
	agentManage := auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "agents.manage", isTokenRevoked)

	r.With(agentRead).Get("/agents/{agentID}/config", func(w http.ResponseWriter, r *http.Request) {
		agentID := strings.TrimSpace(chi.URLParam(r, "agentID"))
		if err := validate.AgentID(agentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		item, err := configs.DesiredInOrganization(r.Context(), orgID, agentID)
		if err != nil {
			if errors.Is(err, store.ErrAgentConfigNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Error(w, "failed to read agent config", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	})

	r.With(agentRead).Get("/agents/{agentID}/config/history", func(w http.ResponseWriter, r *http.Request) {
		agentID := strings.TrimSpace(chi.URLParam(r, "agentID"))
		if err := validate.AgentID(agentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		items, err := configs.HistoryInOrganization(r.Context(), orgID, agentID, 20)
		if err != nil {
			http.Error(w, "failed to read agent config history", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	createConfig := func(w http.ResponseWriter, r *http.Request, agentID string, settings store.AgentConfigSettings, action string, rollbackFrom int64) {
		if signer == nil {
			http.Error(w, "agent remote configuration is not configured", http.StatusServiceUnavailable)
			return
		}
		if err := validateRemoteSettings(settings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if settings.SatelliteID != nil {
			exists, err := satellites.Exists(r.Context(), *settings.SatelliteID)
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
		createdBy := claims.UserID
		item, err := configs.CreateInOrganization(r.Context(), orgID, agentID, settings, &createdBy, func(version int64, value store.AgentConfigSettings, issuedAt time.Time) (store.AgentConfigSignResult, error) {
			envelope, payload, signature, err := signer.Sign(r.Context(), agentID, version, agentconfig.Settings{
				FrequencyHz:            value.FrequencyHz,
				CaptureIntervalSeconds: value.CaptureIntervalSeconds,
				SatelliteID:            value.SatelliteID,
			}, issuedAt)
			if err != nil {
				return store.AgentConfigSignResult{}, err
			}
			return store.AgentConfigSignResult{
				Payload: payload, Signature: signature, Checksum: envelope.Checksum, SigningKID: envelope.KID,
			}, nil
		})
		if err != nil {
			switch {
			case errors.Is(err, store.ErrAgentIdentityNotFound):
				http.Error(w, "agent not found", http.StatusNotFound)
			case errors.Is(err, store.ErrAgentConfigInactive):
				http.Error(w, "only active agents can receive configuration", http.StatusConflict)
			default:
				http.Error(w, "failed to create agent config", http.StatusInternalServerError)
			}
			return
		}
		details := map[string]interface{}{
			"agent_id":                 agentID,
			"version":                  item.Version,
			"frequency_hz":             settings.FrequencyHz,
			"capture_interval_seconds": settings.CaptureIntervalSeconds,
			"satellite_id":             settings.SatelliteID,
			"checksum_sha256":          item.Checksum,
			"signing_kid":              item.SigningKID,
		}
		if rollbackFrom > 0 {
			details["rollback_from_version"] = rollbackFrom
		}
		logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), action, clientIP(r), details)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	}

	r.With(agentManage).Put("/agents/{agentID}/config", func(w http.ResponseWriter, r *http.Request) {
		agentID := strings.TrimSpace(chi.URLParam(r, "agentID"))
		if err := validate.AgentID(agentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var body store.AgentConfigSettings
		if err := decodeAgentJSON(w, r, 4<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		createConfig(w, r, agentID, body, audit.ActionAgentConfigIssued, 0)
	})

	r.With(agentManage).Post("/agents/{agentID}/config/rollback", func(w http.ResponseWriter, r *http.Request) {
		agentID := strings.TrimSpace(chi.URLParam(r, "agentID"))
		if err := validate.AgentID(agentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var body struct {
			Version int64 `json:"version"`
		}
		if err := decodeAgentJSON(w, r, 2<<10, &body); err != nil || body.Version <= 0 {
			http.Error(w, "invalid rollback version", http.StatusBadRequest)
			return
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		settings, err := configs.SettingsForVersionInOrganization(r.Context(), orgID, agentID, body.Version)
		if err != nil {
			if errors.Is(err, store.ErrAgentConfigNotFound) {
				http.Error(w, "config version not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to read config version", http.StatusInternalServerError)
			return
		}
		createConfig(w, r, agentID, settings, audit.ActionAgentConfigRollback, body.Version)
	})

	r.Get("/agents/me/config", func(w http.ResponseWriter, r *http.Request) {
		if signer == nil {
			http.Error(w, "agent remote configuration is not configured", http.StatusServiceUnavailable)
			return
		}
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
			return
		}
		fingerprintHash := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
		fingerprint := hex.EncodeToString(fingerprintHash[:])
		agentID, err := identities.AgentIDForCertificate(r.Context(), fingerprint)
		if err != nil {
			http.Error(w, "invalid agent certificate", http.StatusUnauthorized)
			return
		}
		orgID, err := identities.OrganizationForAgent(r.Context(), agentID)
		if err != nil {
			http.Error(w, "agent organization unavailable", http.StatusUnauthorized)
			return
		}
		if !enforceAgentRateLimit(w, r, tenantLimiter, orgID, agentID) {
			return
		}
		afterVersion := int64(0)
		if raw := strings.TrimSpace(r.URL.Query().Get("after_version")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || parsed < 0 {
				http.Error(w, "after_version must be a non-negative integer", http.StatusBadRequest)
				return
			}
			afterVersion = parsed
		}
		item, err := configs.Desired(r.Context(), agentID)
		if err != nil {
			if errors.Is(err, store.ErrAgentConfigNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Error(w, "failed to read agent config", http.StatusInternalServerError)
			return
		}
		if item.Version <= afterVersion {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if item.SigningKID != "" && signingKeys != nil {
			if _, err := signingKeys.SigningKey(r.Context(), item.SigningKID); err != nil {
				http.Error(w, "desired config signing key is no longer trusted", http.StatusConflict)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agentconfig.Envelope{
			KID:       item.SigningKID,
			Payload:   base64.RawURLEncoding.EncodeToString(item.Payload),
			Signature: base64.RawURLEncoding.EncodeToString(item.Signature),
			Checksum:  item.Checksum,
		})
	})

	r.Post("/agents/me/config/ack", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "mTLS client certificate required", http.StatusUnauthorized)
			return
		}
		fingerprintHash := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
		fingerprint := hex.EncodeToString(fingerprintHash[:])
		agentID, err := identities.AgentIDForCertificate(r.Context(), fingerprint)
		if err != nil {
			http.Error(w, "invalid agent certificate", http.StatusUnauthorized)
			return
		}
		var body struct {
			Version  int64  `json:"version"`
			Status   string `json:"status"`
			Message  string `json:"message"`
			Checksum string `json:"checksum_sha256"`
		}
		if err := decodeAgentJSON(w, r, 4<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		body.Status = strings.ToLower(strings.TrimSpace(body.Status))
		body.Message = strings.TrimSpace(body.Message)
		body.Checksum = strings.ToLower(strings.TrimSpace(body.Checksum))
		if body.Version <= 0 || (body.Status != "applied" && body.Status != "rejected") || len(body.Message) > 256 || len(body.Checksum) != 64 {
			http.Error(w, "invalid acknowledgement", http.StatusBadRequest)
			return
		}
		if err := configs.Ack(r.Context(), agentID, body.Version, body.Status, body.Message, body.Checksum); err != nil {
			if errors.Is(err, store.ErrAgentConfigNotFound) {
				http.Error(w, "config version not found", http.StatusNotFound)
				return
			}
			if errors.Is(err, store.ErrAgentConfigAckConflict) {
				http.Error(w, "acknowledgement conflict", http.StatusConflict)
				return
			}
			http.Error(w, "failed to acknowledge config", http.StatusInternalServerError)
			return
		}
		logAudit(r.Context(), auditLog, audit.ActorAgent, agentID, audit.ActionAgentConfigAcknowledged, clientIP(r), map[string]interface{}{
			"version": body.Version,
			"status":  body.Status,
		})
		w.WriteHeader(http.StatusNoContent)
	})
}
