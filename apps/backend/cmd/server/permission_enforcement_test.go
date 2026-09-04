package main

import (
	"os"
	"strings"
	"testing"
)

func TestSensitiveRoutesUseCapabilities(t *testing.T) {
	joined := readServerProductionSource(t)
	for _, permission := range []string{
		"users.read", "users.manage", "agents.read", "agents.manage",
		"assets.read", "assets.manage", "incidents.read", "incidents.manage",
		"audit.read", "programs.read", "programs.manage", "rf.read", "rf.ingest",
		"orbital.read", "orbital.manage", "organizations.read", "rbac.manage",
	} {
		if !strings.Contains(joined, permission) {
			t.Fatalf("missing permission enforcement %s", permission)
		}
	}
	organizationSource, err := os.ReadFile("organizations.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"IsPlatformAdministrator", "requirePlatformAdmin"} {
		if !strings.Contains(string(organizationSource), token) {
			t.Fatalf("missing platform administrator enforcement %s", token)
		}
	}
}
