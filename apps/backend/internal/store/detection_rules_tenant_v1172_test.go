package store

import (
	"os"
	"strings"
	"testing"
)

func TestDetectionCooldownIntegrationUsesActorOrganizationV1172(t *testing.T) {
	data, err := os.ReadFile("integration_test.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	start := strings.Index(src, "func TestDetectionRuleMatchCooldown")
	if start < 0 {
		t.Fatal("cooldown integration test missing")
	}
	end := strings.Index(src[start:], "\nfunc ")
	if end < 0 {
		end = len(src) - start
	}
	block := src[start : start+end]
	for _, token := range []string{
		"organizationID := actor.OrganizationID",
		"CreateInOrganization(ctx, organizationID",
		"OrganizationID: organizationID",
		"RecordMatchIfAllowed(ctx, organizationID",
	} {
		if !strings.Contains(block, token) {
			t.Fatalf("tenant-aware cooldown test missing %q", token)
		}
	}
	if strings.Contains(block, "DefaultOrganizationID") {
		t.Fatal("cooldown integration test must not depend on the default tenant")
	}
}
