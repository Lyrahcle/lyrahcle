package store

import (
	"os"
	"strings"
	"testing"
)

func TestRBACPermissionQueriesBindRoleToOrganization(t *testing.T) {
	data, err := os.ReadFile("rbac.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if strings.Count(src, "JOIN organization_roles r ON r.id=umr.organization_role_id AND r.organization_id=umr.organization_id") < 2 {
		t.Fatal("RBAC permission queries must bind assigned role to membership organization")
	}
}
