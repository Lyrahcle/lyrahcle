package main

import (
	"os"
	"strings"
	"testing"
)

func TestPlatformUIHasDiscoverableAccessContract(t *testing.T) {
	data, err := os.ReadFile("organizations.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{`Get("/me"`, `"platform_admin"`, "IsPlatformAdministrator"} {
		if !strings.Contains(src, token) {
			t.Fatalf("platform UI access contract missing %q", token)
		}
	}
}
