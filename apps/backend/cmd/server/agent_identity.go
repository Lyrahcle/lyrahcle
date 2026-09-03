package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mlko01/lyrahcle/apps/backend/internal/agentidentity"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

func decodeAgentJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst interface{}) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func generateEnrollmentToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	plaintext := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plaintext))
	return plaintext, hex.EncodeToString(hash[:]), nil
}

func hashEnrollmentToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func registerAgentIdentityRoutes(
	r chi.Router,
	cfg config.Config,
	ca *agentidentity.CA,
	identities *store.AgentIdentityStore,
	agentKeys *store.AgentKeyStore,
	auditLog *store.AuditStore,
	rbac *store.RBACStore,
	isTokenRevoked auth.RevocationCheck,
	tenantLimiter *ratelimit.Limiter,
) {
	r.Post("/agents/enroll", func(w http.ResponseWriter, r *http.Request) {
		if ca == nil {
			http.Error(w, "agent enrollment is not configured", http.StatusServiceUnavailable)
			return
		}
		if cfg.IsProduction() && r.TLS == nil {
			http.Error(w, "TLS is required for agent enrollment", http.StatusUpgradeRequired)
			return
		}
		var body struct {
			Token           string `json:"token"`
			CSRPEM          string `json:"csr_pem"`
			AgentVersion    string `json:"agent_version"`
			FirmwareVersion string `json:"firmware_version"`
		}
		if err := decodeAgentJSON(w, r, 32<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		body.Token = strings.TrimSpace(body.Token)
		body.AgentVersion = strings.TrimSpace(body.AgentVersion)
		body.FirmwareVersion = strings.TrimSpace(body.FirmwareVersion)
		if len(body.Token) < 32 || len(body.Token) > 128 || len(body.CSRPEM) > 24<<10 || len(body.AgentVersion) > 64 || len(body.FirmwareVersion) > 128 {
			http.Error(w, "invalid enrollment payload", http.StatusBadRequest)
			return
		}
		tokenHash := hashEnrollmentToken(body.Token)
		agentID, err := identities.AgentIDForEnrollmentToken(r.Context(), tokenHash)
		if err != nil {
			http.Error(w, "invalid or expired enrollment token", http.StatusUnauthorized)
			return
		}
		issued, err := ca.SignCSR(agentID, body.CSRPEM, cfg.AgentCertificateTTL)
		if err != nil {
			http.Error(w, "invalid certificate signing request", http.StatusBadRequest)
			return
		}
		cert := store.AgentCertificate{
			SerialNumber:      issued.SerialNumber,
			FingerprintSHA256: issued.FingerprintSHA256,
			CertificatePEM:    issued.PEM,
			NotBefore:         issued.NotBefore,
			NotAfter:          issued.NotAfter,
		}
		consumedAgentID, err := identities.ConsumeEnrollmentToken(r.Context(), tokenHash, cert, body.AgentVersion, body.FirmwareVersion)
		if err != nil {
			if errors.Is(err, store.ErrAgentEnrollmentInvalid) || errors.Is(err, store.ErrAgentIdentityInactive) {
				http.Error(w, "invalid or expired enrollment token", http.StatusUnauthorized)
				return
			}
			http.Error(w, "agent enrollment failed", http.StatusInternalServerError)
			return
		}
		logAudit(r.Context(), auditLog, audit.ActorAgent, consumedAgentID, audit.ActionAgentEnrolled, clientIP(r), map[string]interface{}{
			"fingerprint_sha256": issued.FingerprintSHA256,
			"not_after":          issued.NotAfter,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agent_id":        consumedAgentID,
			"certificate_pem": issued.PEM,
			"ca_pem":          ca.CertificatePEM(),
			"expires_at":      issued.NotAfter,
		})
	})

	certLookup := func(ctx context.Context, fingerprint string) (string, error) {
		agentID, err := identities.AgentIDForCertificate(ctx, fingerprint)
		if err != nil {
			logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentCertificateAuthFailed, "", nil)
		}
		return agentID, err
	}
	keyLookup := func(ctx context.Context, keyHash string) (string, error) {
		agentID, err := agentKeys.AgentIDForHash(ctx, keyHash)
		if err != nil {
			logAudit(ctx, auditLog, audit.ActorAnonymous, "unknown", audit.ActionAgentKeyAuthFailed, "", nil)
		}
		return agentID, err
	}
	credentialAuth := auth.RequireAgentCredential(keyLookup, certLookup)
	agentRead := auth.RequireAnyPermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, []string{"agents.read", "agents.manage"}, isTokenRevoked)
	agentManage := auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "agents.manage", isTokenRevoked)

	r.With(agentRead).Get("/agents/", func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		items, err := identities.ListByOrganization(r.Context(), orgID)
		if err != nil {
			http.Error(w, "failed to list agents", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	r.With(agentManage).Post("/agents/enrollment-tokens", func(w http.ResponseWriter, r *http.Request) {
		if ca == nil {
			http.Error(w, "agent enrollment is not configured", http.StatusServiceUnavailable)
			return
		}
		var body struct {
			AgentID string `json:"agent_id"`
		}
		if err := decodeAgentJSON(w, r, 4<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		body.AgentID = strings.TrimSpace(body.AgentID)
		if err := validate.AgentID(body.AgentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		plaintext, tokenHash, err := generateEnrollmentToken()
		if err != nil {
			http.Error(w, "failed to generate enrollment token", http.StatusInternalServerError)
			return
		}
		expiresAt := time.Now().UTC().Add(cfg.AgentEnrollmentTTL)
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		if err := identities.CreateEnrollmentTokenInOrganization(r.Context(), orgID, body.AgentID, tokenHash, expiresAt); err != nil {
			if errors.Is(err, store.ErrAgentAlreadyEnrolled) {
				http.Error(w, "agent is already enrolled; rotate its certificate instead", http.StatusConflict)
				return
			}
			if errors.Is(err, store.ErrAgentIdentityInactive) {
				http.Error(w, "inactive agents cannot be enrolled", http.StatusConflict)
				return
			}
			http.Error(w, "failed to create enrollment token", http.StatusInternalServerError)
			return
		}
		if claims, ok := auth.FromContext(r.Context()); ok {
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentEnrollmentIssued, clientIP(r), map[string]interface{}{
				"agent_id":   body.AgentID,
				"expires_at": expiresAt,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agent_id":   body.AgentID,
			"token":      plaintext,
			"expires_at": expiresAt,
		})
	})

	r.With(agentManage).Put("/agents/{agentID}/station", func(w http.ResponseWriter, r *http.Request) {
		agentID := strings.TrimSpace(chi.URLParam(r, "agentID"))
		if err := validate.AgentID(agentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var body store.AgentStation
		if err := decodeAgentJSON(w, r, 2<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if body.LatitudeDeg < -90 || body.LatitudeDeg > 90 || body.LongitudeDeg < -180 || body.LongitudeDeg > 180 || body.AltitudeM < -500 || body.AltitudeM > 10000 {
			http.Error(w, "invalid station coordinates", http.StatusBadRequest)
			return
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		if err := identities.SetStationInOrganization(r.Context(), orgID, agentID, body); err != nil {
			if errors.Is(err, store.ErrAgentIdentityNotFound) {
				http.Error(w, "agent not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to update station", http.StatusInternalServerError)
			return
		}
		if claims, ok := auth.FromContext(r.Context()); ok {
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentStationUpdated, clientIP(r), map[string]interface{}{
				"agent_id": agentID, "latitude_deg": body.LatitudeDeg, "longitude_deg": body.LongitudeDeg, "altitude_m": body.AltitudeM,
			})
		}
		w.WriteHeader(http.StatusNoContent)
	})

	r.With(agentManage).Patch("/agents/{agentID}/status", func(w http.ResponseWriter, r *http.Request) {
		agentID := strings.TrimSpace(chi.URLParam(r, "agentID"))
		if err := validate.AgentID(agentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var body struct {
			Status string `json:"status"`
		}
		if err := decodeAgentJSON(w, r, 2<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		body.Status = strings.ToLower(strings.TrimSpace(body.Status))
		if body.Status != "active" && body.Status != "quarantined" && body.Status != "revoked" {
			http.Error(w, "status must be active, quarantined, or revoked", http.StatusBadRequest)
			return
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		if err := identities.SetStatusInOrganization(r.Context(), orgID, agentID, body.Status); err != nil {
			if errors.Is(err, store.ErrAgentIdentityNotFound) {
				http.Error(w, "agent not found", http.StatusNotFound)
				return
			}
			if errors.Is(err, store.ErrAgentStatusTransition) {
				http.Error(w, "invalid agent status transition", http.StatusConflict)
				return
			}
			http.Error(w, "failed to update agent status", http.StatusInternalServerError)
			return
		}
		if claims, ok := auth.FromContext(r.Context()); ok {
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionAgentStatusChanged, clientIP(r), map[string]interface{}{
				"agent_id": agentID,
				"status":   body.Status,
			})
		}
		w.WriteHeader(http.StatusNoContent)
	})

	r.With(credentialAuth).Post("/agents/me/certificate/rotate", func(w http.ResponseWriter, r *http.Request) {
		if ca == nil {
			http.Error(w, "agent certificate rotation is not configured", http.StatusServiceUnavailable)
			return
		}
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
			return
		}
		agentID, ok := auth.AgentIDFromContext(r.Context())
		if !ok {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
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
		var body struct {
			CSRPEM string `json:"csr_pem"`
		}
		if err := decodeAgentJSON(w, r, 24<<10, &body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		issued, err := ca.SignCSR(agentID, body.CSRPEM, cfg.AgentCertificateTTL)
		if err != nil {
			http.Error(w, "invalid certificate signing request", http.StatusBadRequest)
			return
		}
		currentFingerprint := agentidentity.Fingerprint(r.TLS.PeerCertificates[0])
		cert := store.AgentCertificate{
			SerialNumber: issued.SerialNumber, FingerprintSHA256: issued.FingerprintSHA256,
			CertificatePEM: issued.PEM, NotBefore: issued.NotBefore, NotAfter: issued.NotAfter,
		}
		if err := identities.RotateCertificate(r.Context(), agentID, currentFingerprint, cert); err != nil {
			http.Error(w, "certificate rotation failed", http.StatusConflict)
			return
		}
		logAudit(r.Context(), auditLog, audit.ActorAgent, agentID, audit.ActionAgentCertificateRotated, clientIP(r), map[string]interface{}{
			"old_fingerprint_sha256": currentFingerprint,
			"new_fingerprint_sha256": issued.FingerprintSHA256,
			"not_after":              issued.NotAfter,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"certificate_pem": issued.PEM,
			"ca_pem":          ca.CertificatePEM(),
			"expires_at":      issued.NotAfter,
		})
	})
}
