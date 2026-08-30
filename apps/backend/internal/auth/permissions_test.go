package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type permissionCheckerFunc func(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)

func (f permissionCheckerFunc) HasPermission(ctx context.Context, u, o uuid.UUID, p string) (bool, error) {
	return f(ctx, u, o, p)
}

func TestRequirePermission(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	checker := permissionCheckerFunc(func(_ context.Context, u, o uuid.UUID, p string) (bool, error) {
		return u == userID && o == orgID && p == "assets.read", nil
	})
	h := RequirePermission(checker, "assets.read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithClaims(req.Context(), &Claims{UserID: userID, OrganizationID: orgID}))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestRequireAnyPermissionAllowsManageOnReadSurface(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	checker := permissionCheckerFunc(func(_ context.Context, u, o uuid.UUID, p string) (bool, error) {
		return u == userID && o == orgID && p == "agents.manage", nil
	})
	h := RequireAnyPermission(checker, "agents.read", "agents.manage")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithClaims(req.Context(), &Claims{UserID: userID, OrganizationID: orgID}))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestRequireAnyPermissionDeniesWithoutMatchingCapability(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	checker := permissionCheckerFunc(func(_ context.Context, _, _ uuid.UUID, _ string) (bool, error) {
		return false, nil
	})
	h := RequireAnyPermission(checker, "agents.read", "agents.manage")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithClaims(req.Context(), &Claims{UserID: userID, OrganizationID: orgID}))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", res.Code, http.StatusForbidden)
	}
}
