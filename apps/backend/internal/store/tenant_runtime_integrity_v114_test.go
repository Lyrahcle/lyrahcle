package store

import (
	"os"
	"strings"
	"testing"
)

func TestIncidentCreateReturnsOrganizationOwnership(t *testing.T) {
	data, err := os.ReadFile("incidents.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "RETURNING id, organization_id, reference") {
		t.Fatal("incident creation must return organization_id")
	}
	if !strings.Contains(src, "&incident.ID, &incident.OrganizationID") {
		t.Fatal("incident creation must scan organization_id into returned incident")
	}
}

func TestLegacyAuditInsertIsExplicitlyDefaultTenant(t *testing.T) {
	data, err := os.ReadFile("audit.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "return s.InsertInOrganization(ctx, DefaultOrganizationID") {
		t.Fatal("legacy audit insert must delegate through explicit tenant-aware insert")
	}
}

func TestGlobalAssetMatchingCarriesOrganization(t *testing.T) {
	data, err := os.ReadFile("assets.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "SELECT id, organization_id, program_id, kind, value FROM assets") {
		t.Fatal("background asset matching must carry organization_id")
	}
}
