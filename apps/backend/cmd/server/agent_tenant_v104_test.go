package main

import (
	"os"
	"strings"
	"testing"
)

func TestAgentAdminRoutesUseTenantScope(t *testing.T) {
	for _, file := range []string{"agent_identity_v90.go", "agent_config_v91.go", "main.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		if file == "agent_identity_v90.go" {
			for _, token := range []string{"ListByOrganization", "CreateEnrollmentTokenInOrganization", "SetStationInOrganization", "SetStatusInOrganization"} {
				if !strings.Contains(src, token) {
					t.Fatalf("%s missing %s", file, token)
				}
			}
		}
		if file == "agent_config_v91.go" && !strings.Contains(src, "CreateInOrganization") {
			t.Fatalf("%s missing scoped config creation", file)
		}
	}
}
