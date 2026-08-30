package main

import (
	"os"
	"strings"
	"testing"
)

func TestCorrelationInputsCarryOrganizationContext(t *testing.T) {
	files := []string{"main.go", "cross_domain_v94.go"}
	joined := ""
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		joined += string(data)
	}
	if !strings.Contains(joined, "OrganizationID") && !strings.Contains(joined, "organizationID") {
		t.Fatal("correlation/background paths have no organization context")
	}
}

func TestIncidentStreamUsesScopedIncidentList(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "incidents.ListByOrganization") {
		t.Fatal("incident SSE/list path is not tenant scoped")
	}
}
