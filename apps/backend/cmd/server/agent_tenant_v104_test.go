package main

import (
	"os"
	"strings"
	"testing"
)

func TestAgentAdminRoutesUseTenantScope(t *testing.T) {
	for _, file := range []string{"agent_identity.go", "agent_config.go", "main.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		if file == "agent_identity.go" {
			for _, token := range []string{"ListByOrganization", "CreateEnrollmentTokenInOrganization", "SetStationInOrganization", "SetStatusInOrganization"} {
				if !strings.Contains(src, token) {
					t.Fatalf("%s missing %s", file, token)
				}
			}
		}
		if file == "agent_config.go" && !strings.Contains(src, "CreateInOrganization") {
			t.Fatalf("%s missing scoped config creation", file)
		}
	}
}
