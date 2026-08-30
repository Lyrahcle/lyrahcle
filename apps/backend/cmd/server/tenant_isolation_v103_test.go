package main

import (
	"os"
	"strings"
	"testing"
)

func TestCriticalHandlersUseTenantScopedStores(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)

	required := []string{
		"assets.ListByOrganization",
		"assets.UpsertInOrganization",
		"auditLog.ListByOrganization",
		"incidents.ListByOrganization",
		"incidents.GetByOrganization",
		"incidents.UpdateByOrganization",
		"users.GetByIDInOrganization",
	}
	for _, token := range required {
		if !strings.Contains(src, token) {
			t.Fatalf("main.go missing tenant-scoped handler call %q", token)
		}
	}
}
