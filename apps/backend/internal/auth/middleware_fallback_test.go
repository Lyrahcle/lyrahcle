package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// RequireAnyRoleWithFallback must allow a request whose token role is one of
// the explicitly allowed roles, even when that role is not the "minimum"
// role used internally to authenticate the token itself.
func TestRequireAnyRoleWithFallbackAllowsListedRole(t *testing.T) {
	secret := []byte("secret")
	token, _ := IssueToken(secret, uuid.New(), RoleAuditor, time.Hour)

	handler := RequireAnyRoleWithFallback(secret, nil, []Role{RoleAuditor, RoleAdmin})(newTestHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// RequireAnyRoleWithFallback must reject a role that is not in the allowlist
// even though it would otherwise satisfy the underlying RoleViewer minimum.
func TestRequireAnyRoleWithFallbackRejectsUnlistedRole(t *testing.T) {
	secret := []byte("secret")
	token, _ := IssueToken(secret, uuid.New(), RoleViewer, time.Hour)

	handler := RequireAnyRoleWithFallback(secret, nil, []Role{RoleAuditor, RoleAdmin})(newTestHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// RequireAnyRoleWithFallback must reject requests with no bearer token at
// all before it ever gets to evaluate the allowlist.
func TestRequireAnyRoleWithFallbackRejectsMissingToken(t *testing.T) {
	handler := RequireAnyRoleWithFallback([]byte("secret"), nil, []Role{RoleAdmin})(newTestHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// RequireAgentKeyOrRoleWithFallback must accept a valid X-Agent-Key without
// ever consulting the JWT secret.
func TestRequireAgentKeyOrRoleWithFallbackAcceptsAgentKey(t *testing.T) {
	plainKey := "lyr_agent_testkey_fallback"
	lookup := fakeLookup(map[string]string{HashAgentKey(plainKey): "sdr-agent-sevilla"})

	handler := RequireAgentKeyOrRoleWithFallback([]byte("current"), []byte("previous"), RoleOperator, lookup)(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Agent-Key", plainKey)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// RequireAgentKeyOrRoleWithFallback must fall back to validating the JWT
// against the previous secret when the current secret does not verify it,
// mirroring the zero-downtime signing-key rotation contract used elsewhere.
func TestRequireAgentKeyOrRoleWithFallbackAcceptsPreviousSecret(t *testing.T) {
	current := []byte("current-secret")
	previous := []byte("previous-secret")
	token, _ := IssueToken(previous, uuid.New(), RoleAdmin, time.Hour)

	handler := RequireAgentKeyOrRoleWithFallback(current, previous, RoleOperator, fakeLookup(nil))(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// RequireAgentKeyOrRoleWithFallback must reject requests presenting neither
// a valid agent key nor a valid bearer token under either secret.
func TestRequireAgentKeyOrRoleWithFallbackRejectsInvalidToken(t *testing.T) {
	handler := RequireAgentKeyOrRoleWithFallback([]byte("current"), []byte("previous"), RoleOperator, fakeLookup(nil))(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
