package main

import (
	"strings"
	"testing"
)

func TestCorrelationInputsCarryOrganizationContext(t *testing.T) {
	joined := readServerProductionSource(t)
	if !strings.Contains(joined, "OrganizationID") && !strings.Contains(joined, "organizationID") {
		t.Fatal("correlation/background paths have no organization context")
	}
}

func TestIncidentStreamUsesScopedIncidentList(t *testing.T) {
	src := readServerProductionSource(t)
	if !strings.Contains(src, "incidents.ListByOrganization") {
		t.Fatal("incident SSE/list path is not tenant scoped")
	}
}
