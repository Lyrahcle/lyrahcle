package main

import (
	"os"
	"strings"
	"testing"
)

func TestDetectionAndOperationalPermissionContractsV117(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		`ListByOrganization(r.Context(), orgID`,
		`GetByOrganization(r.Context(), orgID, ruleID)`,
		`CreateInOrganization(r.Context(), orgID`,
		`UpdateInOrganization(r.Context(), orgID`,
		`"detections.read"`,
		`"detections.manage"`,
		`"alerts.read"`,
		`"alerts.manage"`,
		`ListEnabledByOrganization(ctx, anomalyRecord.OrganizationID)`,
		`AttachIncidentInOrganization(ctx, anomalyRecord.OrganizationID`,
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("v11.7 security contract missing %q", token)
		}
	}
	if !strings.Contains(src, `RequirePermission(rbac, "orbital.read")).Get("/{satelliteID}/passes"`) {
		t.Fatal("satellite pass prediction must require orbital.read")
	}
}

func TestDetectionMigrationScopesRulesByOrganizationV117(t *testing.T) {
	data, err := os.ReadFile("../../migrations/0034_detection_rbac_completion.sql")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"ALTER TABLE rf_detection_rules ADD COLUMN IF NOT EXISTS organization_id",
		"idx_rf_detection_rules_org_name",
		"ALTER TABLE rf_detection_matches ADD COLUMN IF NOT EXISTS organization_id",
		"'detections.read'",
		"'detections.manage'",
		"'alerts.read'",
		"'alerts.manage'",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("v11.7 migration missing %q", token)
		}
	}
}
