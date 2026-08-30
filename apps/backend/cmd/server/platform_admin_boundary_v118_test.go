package main

import (
	"os"
	"strings"
	"testing"
)

func TestOrganizationRoutesUsePlatformAdminBoundary(t *testing.T) {
	data, err := os.ReadFile("organizations_v100.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if strings.Contains(src, `"organizations.manage"`) || strings.Contains(src, `"organizations.read"`) {
		t.Fatal("global organization routes must not rely on tenant-local organization permissions")
	}
	for _, token := range []string{
		"IsPlatformAdministrator",
		"platform administrator required",
		"requirePlatformAdmin",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("platform admin boundary missing %q", token)
		}
	}
}

func TestPlatformAdminMigrationExists(t *testing.T) {
	data, err := os.ReadFile("../../migrations/0035_platform_admin_boundary.sql")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"CREATE TABLE IF NOT EXISTS platform_administrators",
		"PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE",
		"00000000-0000-0000-0000-000000000001",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("platform admin migration missing %q", token)
		}
	}
}
