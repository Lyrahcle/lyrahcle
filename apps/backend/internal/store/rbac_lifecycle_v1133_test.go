package store

import (
	"os"
	"strings"
	"testing"
)

func TestRBACLifecycleMigrationCoversNewOrganizationsAndUsers(t *testing.T) {
	data, err := os.ReadFile("../../migrations/0033_rbac_lifecycle_integrity.sql")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"organizations_seed_rbac",
		"users_sync_compatibility_role",
		"lyrahcle_seed_organization_rbac",
		"organization_membership_roles_user_org_fkey",
		"organization_membership_roles_role_org_fkey",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("RBAC lifecycle migration missing %q", token)
		}
	}
}
