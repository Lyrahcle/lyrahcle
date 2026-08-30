package main

import (
	"os"
	"strings"
	"testing"
)

func TestPlatformAdminLifecycleIsAvailableToUI(t *testing.T) {
	data, err := os.ReadFile("organizations_v100.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{`Get("/platform-users"`, `Post("/platform-admins/{userID}"`, `Delete("/platform-admins/{userID}"`, "self-revocation is not allowed"} {
		if !strings.Contains(src, token) {
			t.Fatalf("platform admin lifecycle missing %q", token)
		}
	}
}
