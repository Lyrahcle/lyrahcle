package main

import (
	"os"
	"strings"
	"testing"
)

func TestRemainingOperationalRoutesUseTenantScope(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"SearchByOrganization", "ListByOrganization(r.Context(), orgID)", "CreateInOrganization",
		"RecentByOrganization", "RecentForAgentInOrganization", "ListByOrganization(r.Context(), orgID, limit, offset)",
		"AcknowledgeInOrganization",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("main.go missing %q", token)
		}
	}
}
