package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
)

func issueRateLimitToken(t *testing.T, secret []byte, orgID, userID uuid.UUID) string {
	t.Helper()
	token, _, err := auth.IssueOrganizationUserTokenWithState(secret, userID, orgID, auth.RoleViewer, false, 1, time.Hour)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	claims, err := auth.ParseToken(secret, token)
	if err != nil {
		t.Fatalf("issued rate-limit token must be valid: %v", err)
	}
	if claims.OrganizationID != orgID || claims.UserID != userID {
		t.Fatalf("issued rate-limit token claims mismatch: org=%s user=%s", claims.OrganizationID, claims.UserID)
	}
	return token
}

func TestAuthenticatedTenantRateLimitIsolatesNoisyNeighbor(t *testing.T) {
	secret := []byte("tenant-rate-limit-test-secret-32bytes-minimum")
	limiter, err := ratelimit.NewTenantLimiter(nil, ratelimit.Policy{
		Window:          time.Minute,
		TenantReadLimit: 2, ActorReadLimit: 2,
		TenantWriteLimit: 2, ActorWriteLimit: 2,
		TenantAgentLimit: 2, AgentLimit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	orgA, orgB := uuid.New(), uuid.New()
	tokenA := issueRateLimitToken(t, secret, orgA, uuid.New())
	tokenB := issueRateLimitToken(t, secret, orgB, uuid.New())

	handler := authenticatedTenantRateLimit(secret, nil, limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	do := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if got := do(tokenA).Code; got != http.StatusNoContent {
		t.Fatalf("tenant A request 1 = %d", got)
	}
	if got := do(tokenA).Code; got != http.StatusNoContent {
		t.Fatalf("tenant A request 2 = %d", got)
	}
	blocked := do(tokenA)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("tenant A request 3 = %d, want 429", blocked.Code)
	}
	if blocked.Header().Get("Retry-After") == "" || blocked.Header().Get("X-RateLimit-Scope") != "tenant" {
		t.Fatalf("missing actionable rate-limit headers: %#v", blocked.Header())
	}
	if got := do(tokenB).Code; got != http.StatusNoContent {
		t.Fatalf("tenant B was throttled by tenant A: %d", got)
	}
}

func TestAuthenticatedTenantRateLimitSeparatesReadAndWrite(t *testing.T) {
	secret := []byte("tenant-rate-limit-test-secret-32bytes-minimum")
	limiter, err := ratelimit.NewTenantLimiter(nil, ratelimit.Policy{
		Window:          time.Minute,
		TenantReadLimit: 5, ActorReadLimit: 5,
		TenantWriteLimit: 1, ActorWriteLimit: 1,
		TenantAgentLimit: 2, AgentLimit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	token := issueRateLimitToken(t, secret, uuid.New(), uuid.New())
	handler := authenticatedTenantRateLimit(secret, nil, limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	do := func(method string) int {
		req := httptest.NewRequest(method, "/api/v1/assets/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}
	if got := do(http.MethodPost); got != http.StatusNoContent {
		t.Fatalf("write 1 = %d", got)
	}
	if got := do(http.MethodPost); got != http.StatusTooManyRequests {
		t.Fatalf("write 2 = %d, want 429", got)
	}
	if got := do(http.MethodGet); got != http.StatusNoContent {
		t.Fatalf("read should have independent budget, got %d", got)
	}
}

func TestInvalidBearerIsLeftForAuthenticationMiddleware(t *testing.T) {
	limiter, err := ratelimit.NewTenantLimiter(nil, ratelimit.Policy{
		Window:          time.Minute,
		TenantReadLimit: 2, ActorReadLimit: 2,
		TenantWriteLimit: 2, ActorWriteLimit: 2,
		TenantAgentLimit: 2, AgentLimit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := authenticatedTenantRateLimit([]byte("secret"), nil, limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("rate limiter usurped authentication responsibility: %d", rec.Code)
	}
}
