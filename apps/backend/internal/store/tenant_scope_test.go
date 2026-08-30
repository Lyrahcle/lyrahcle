package store

import (
	"os"
	"strings"
	"testing"
)

func TestCriticalTenantScopeQueriesIncludeOrganization(t *testing.T) {
	data, err := os.ReadFile("tenant_scope.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)

	required := []string{
		"FROM agent_identities WHERE agent_id=$1 AND organization_id=$2",
		"FROM assets WHERE id=$1 AND organization_id=$2",
		"FROM incidents WHERE id=$1 AND organization_id=$2",
		"FROM agent_identities WHERE organization_id=$1",
		"FROM assets WHERE organization_id=$1",
		"FROM incidents WHERE organization_id=$1",
		"FROM audit_log WHERE organization_id=$1",
	}
	for _, query := range required {
		if !strings.Contains(src, query) {
			t.Errorf("missing organization-scoped tenant query %q", query)
		}
	}
}
