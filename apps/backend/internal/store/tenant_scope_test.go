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
	for _, table := range []string{"agents", "assets", "incidents", "audit_log"} {
		if !strings.Contains(src, "FROM "+table+" WHERE") {
			t.Fatalf("missing scoped query for %s", table)
		}
	}
	if strings.Count(src, "organization_id=$") < 8 {
		t.Fatalf("critical tenant queries are not consistently organization scoped")
	}
}
