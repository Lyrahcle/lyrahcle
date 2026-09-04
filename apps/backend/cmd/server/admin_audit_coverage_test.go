package main

import (
	"os"
	"strings"
	"testing"
)

func TestRBACAdministrativeMutationsAreAudited(t *testing.T) {
	data, err := os.ReadFile("rbac.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"audit.ActionRBACRoleCreated",
		"audit.ActionRBACRoleAssigned",
		"audit.ActionRBACRoleRemoved",
		"logAuditInOrganization",
		`"target_user_id"`,
		`"role_id"`,
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("RBAC administrative audit contract missing %q", token)
		}
	}
}

func TestRBACReadEndpointsUseReadCapability(t *testing.T) {
	data, err := os.ReadFile("rbac.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if strings.Count(src, `"rbac.read"`) < 2 {
		t.Fatal("RBAC permission and role listings must require rbac.read")
	}
}

func TestOrganizationCreationAuditsTargetOrganization(t *testing.T) {
	data, err := os.ReadFile("organizations.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"logAuditInOrganization(r.Context(), auditLog, org.ID",
		"audit.ActionOrganizationCreated",
		`"actor_organization_id"`,
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("organization audit contract missing %q", token)
		}
	}
}
