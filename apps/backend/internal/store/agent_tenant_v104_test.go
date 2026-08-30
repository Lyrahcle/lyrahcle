package store

import (
	"os"
	"strings"
	"testing"
)

func TestAgentTenantScopedMethodsPresent(t *testing.T) {
	files := []string{"agent_identity.go", "agent_config.go", "agent_keys.go"}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "organization_id") && !strings.Contains(string(data), "Organization") {
			t.Fatalf("%s missing organization scope", file)
		}
	}
}
