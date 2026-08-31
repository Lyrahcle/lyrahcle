package main

import (
	"strings"
	"testing"
)

func TestCriticalHandlersUseTenantScopedStores(t *testing.T) {
	src := readServerProductionSource(t)

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
			t.Fatalf("server route package missing tenant-scoped handler call %q", token)
		}
	}
}
