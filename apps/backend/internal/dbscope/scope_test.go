package dbscope

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRequestAndOrganizationScope(t *testing.T) {
	ctx := context.Background()
	if IsRequest(ctx) {
		t.Fatal("background context must not be treated as an HTTP request")
	}
	ctx = MarkRequest(ctx)
	if !IsRequest(ctx) {
		t.Fatal("request marker missing")
	}
	orgID := uuid.New()
	ctx = WithOrganization(ctx, orgID)
	got, ok := OrganizationID(ctx)
	if !ok || got != orgID {
		t.Fatalf("organization scope = %v, %v; want %s, true", got, ok, orgID)
	}
}

func TestPlatformControlPlaneScopeIsExplicit(t *testing.T) {
	ctx := MarkRequest(context.Background())
	if IsPlatformControlPlane(ctx) {
		t.Fatal("ordinary request must not have platform control-plane scope")
	}
	ctx = WithPlatformControlPlane(ctx)
	if !IsPlatformControlPlane(ctx) {
		t.Fatal("explicit platform control-plane scope missing")
	}
}
