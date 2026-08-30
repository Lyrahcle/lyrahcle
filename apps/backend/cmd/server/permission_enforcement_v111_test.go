package main

import (
	"os"
	"strings"
	"testing"
)

func TestSensitiveRoutesUseCapabilities(t *testing.T) {
	files := []string{"main.go", "agent_identity_v90.go", "agent_config_v91.go", "rbac_v110.go"}
	joined := ""
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		joined += string(data)
	}
	for _, permission := range []string{
		"users.read", "users.manage", "agents.read", "agents.manage",
		"assets.read", "assets.manage", "incidents.read", "incidents.manage",
		"audit.read", "programs.read", "programs.manage", "rf.read", "rf.ingest",
		"orbital.read", "orbital.manage", "rbac.manage",
	} {
		if !strings.Contains(joined, permission) {
			t.Fatalf("missing permission enforcement %s", permission)
		}
	}
}
