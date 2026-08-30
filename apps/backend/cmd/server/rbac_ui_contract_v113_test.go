package main

import (
	"os"
	"strings"
	"testing"
)

func TestRBACUIAssignmentContract(t *testing.T) {
	data, err := os.ReadFile("rbac_v110.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		`Get("/users/{userID}/roles"`,
		`"rbac.read"`,
		`Post("/users/{userID}/roles/{roleID}"`,
		`Delete("/users/{userID}/roles/{roleID}"`,
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("RBAC UI contract missing %q", token)
		}
	}
}
