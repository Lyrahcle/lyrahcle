package dbscope

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestScopeDefaultsAreFailClosed(t *testing.T) {
	ctx := context.Background()
	if IsRequest(ctx) {
		t.Fatal("plain context must not be request scoped")
	}
	if IsPlatformControlPlane(ctx) {
		t.Fatal("plain context must not be platform control plane")
	}
	if org, ok := OrganizationID(ctx); ok || org != uuid.Nil {
		t.Fatalf("OrganizationID=(%s,%v) want nil,false", org, ok)
	}
}

func TestWithOrganizationDoesNotGrantPlatformControlPlane(t *testing.T) {
	org := uuid.New()
	ctx := WithOrganization(context.Background(), org)
	if IsPlatformControlPlane(ctx) {
		t.Fatal("tenant scope must not imply platform control plane")
	}
	got, ok := OrganizationID(ctx)
	if !ok || got != org {
		t.Fatalf("OrganizationID=(%s,%v) want (%s,true)", got, ok, org)
	}
}

func TestPlatformControlPlaneDoesNotInventTenantScope(t *testing.T) {
	ctx := WithPlatformControlPlane(context.Background())
	if !IsPlatformControlPlane(ctx) {
		t.Fatal("control-plane marker missing")
	}
	if _, ok := OrganizationID(ctx); ok {
		t.Fatal("platform scope must not invent organization context")
	}
}

func TestMiddlewareMarksOnlyRequestScope(t *testing.T) {
	var captured context.Context
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Context()
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	Middleware(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
	if !IsRequest(captured) {
		t.Fatal("middleware must mark request context")
	}
	if IsPlatformControlPlane(captured) {
		t.Fatal("middleware must not grant platform scope")
	}
	if _, ok := OrganizationID(captured); ok {
		t.Fatal("middleware must not fabricate tenant scope")
	}
}

func TestExplicitScopeCompositionPreservesBothMarkers(t *testing.T) {
	org := uuid.New()
	ctx := MarkRequest(context.Background())
	ctx = WithOrganization(ctx, org)
	ctx = WithPlatformControlPlane(ctx)
	if !IsRequest(ctx) || !IsPlatformControlPlane(ctx) {
		t.Fatal("explicit markers lost")
	}
	got, ok := OrganizationID(ctx)
	if !ok || got != org {
		t.Fatalf("organization lost: %s %v", got, ok)
	}
}
