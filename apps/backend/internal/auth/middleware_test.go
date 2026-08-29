package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireRoleMissingToken(t *testing.T) {
	handler := RequireRole([]byte("secret"), RoleOperator)(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireRoleInsufficientRole(t *testing.T) {
	secret := []byte("secret")
	token, _ := IssueToken(secret, uuid.New(), RoleViewer, time.Hour)

	handler := RequireRole(secret, RoleOperator)(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRequireRoleSufficientRole(t *testing.T) {
	secret := []byte("secret")
	token, _ := IssueToken(secret, uuid.New(), RoleAdmin, time.Hour)

	handler := RequireRole(secret, RoleOperator)(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func fakeLookup(valid map[string]string) AgentKeyLookup {
	return func(ctx context.Context, keyHash string) (string, error) {
		if agentID, ok := valid[keyHash]; ok {
			return agentID, nil
		}
		return "", errors.New("not found")
	}
}

func TestRequireAgentKeyOrRoleWithValidAgentKey(t *testing.T) {
	plainKey := "lyr_agent_testkey"
	lookup := fakeLookup(map[string]string{HashAgentKey(plainKey): "sdr-agent-madrid"})

	var capturedAgentID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAgentID, _ = AgentIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAgentKeyOrRole([]byte("secret"), RoleOperator, lookup)(inner)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Agent-Key", plainKey)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if capturedAgentID != "sdr-agent-madrid" {
		t.Errorf("agent ID in context = %q, want %q", capturedAgentID, "sdr-agent-madrid")
	}
}

func TestRequireAgentKeyOrRoleWithInvalidAgentKey(t *testing.T) {
	lookup := fakeLookup(map[string]string{})

	handler := RequireAgentKeyOrRole([]byte("secret"), RoleOperator, lookup)(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Agent-Key", "lyr_agent_wrongkey")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireAgentKeyOrRoleFallsBackToJWT(t *testing.T) {
	secret := []byte("secret")
	token, _ := IssueToken(secret, uuid.New(), RoleAdmin, time.Hour)
	lookup := fakeLookup(map[string]string{})

	handler := RequireAgentKeyOrRole(secret, RoleOperator, lookup)(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (should fall back to JWT auth when no agent key header is present)", rec.Code, http.StatusOK)
	}
}

func TestRequireAgentKeyOrRoleNoCredentials(t *testing.T) {
	lookup := fakeLookup(map[string]string{})
	handler := RequireAgentKeyOrRole([]byte("secret"), RoleOperator, lookup)(newTestHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireRoleReturnsServiceUnavailableWhenRevocationCheckFails(t *testing.T) {
	secret := []byte("secret")
	token, err := IssueToken(secret, uuid.New(), RoleAdmin, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	check := func(ctx context.Context, claims *Claims) (bool, error) {
		return false, errors.New("redis unavailable")
	}

	handler := RequireRole(secret, RoleViewer, check)(newTestHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestRequireRoleWithFallbackAcceptsPreviousKey(t *testing.T) {
	current := []byte("current-secret-current-secret-123456")
	previous := []byte("previous-secret-previous-secret-1234")
	token, err := IssueToken(previous, uuid.New(), RoleOperator, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	handler := RequireRoleWithFallback(current, previous, RoleViewer)(newTestHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
