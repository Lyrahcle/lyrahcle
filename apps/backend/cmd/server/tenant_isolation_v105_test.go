package main

import (
	"strings"
	"testing"
)

func TestRemainingOperationalRoutesUseTenantScope(t *testing.T) {
	src := readServerProductionSource(t)
	for _, token := range []string{
		"SearchByOrganization", "ListByOrganization(r.Context(), orgID)", "CreateInOrganization",
		"RecentByOrganization", "RecentForAgentInOrganization", "ListByOrganization(r.Context(), orgID, limit, offset)",
		"AcknowledgeInOrganization",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("server route package missing %q", token)
		}
	}
}
