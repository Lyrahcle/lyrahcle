package store

import (
	"os"
	"strings"
	"testing"
)

func TestRBACQueriesAreOrganizationScoped(t *testing.T) {
	data, err := os.ReadFile("rbac.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"umr.organization_id=$2",
		"organization_roles WHERE organization_id=$1",
		"organization_id=$2",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("rbac.go missing %q", token)
		}
	}
}
