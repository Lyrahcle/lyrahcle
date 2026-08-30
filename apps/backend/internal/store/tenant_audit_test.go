package store

import (
	"os"
	"strings"
	"testing"
)

func TestTenantAuditCoversOwnedDomains(t *testing.T) {
	data, err := os.ReadFile("tenant_audit.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, table := range []string{
		"users",
		"agent_identities",
		"assets",
		"incidents",
		"audit_log",
		"programs",
		"rf_readings",
		"rf_anomalies",
		"orbital_events",
	} {
		if !strings.Contains(src, "FROM "+table+" WHERE organization_id=$1") &&
			!strings.Contains(src, "FROM "+table+" WHERE organization_id=$1 AND") {
			t.Fatalf("tenant snapshot missing %s", table)
		}
	}
}
