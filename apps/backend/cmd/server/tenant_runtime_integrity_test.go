package main

import (
	"strings"
	"testing"
)

func TestAutomatedRFIncidentKeepsTenantOwnership(t *testing.T) {
	src := readServerProductionSource(t)

	for _, token := range []string{
		"OrganizationID:  anomalyRecord.OrganizationID",
		`"organization_id": anomalyRecord.OrganizationID.String()`,
		"auditLog.InsertInOrganization(ctx, organizationID",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("tenant-aware RF correlation/audit contract missing %q", token)
		}
	}
}

func TestAuditHelperUsesAuthenticatedOrganization(t *testing.T) {
	src := readServerProductionSource(t)
	for _, token := range []string{
		"auth.FromContext(ctx)",
		"claims.OrganizationID",
		`details["organization_id"]`,
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("audit tenant derivation missing %q", token)
		}
	}
}
