package store

import (
	"os"
	"strings"
	"testing"
)

func TestUserStoreExposesOrganizationScopedAdministration(t *testing.T) {
	data, err := os.ReadFile("users.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	required := []string{
		"CreateInOrganization",
		"ListByOrganization",
		"GetByIDInOrganization",
		"CountAdminsInOrganization",
		"UpdateRoleInOrganization",
		"SetEnabledInOrganization",
		"SoftDeleteInOrganization",
		"organization_id = $",
	}
	for _, token := range required {
		if !strings.Contains(src, token) {
			t.Fatalf("users.go missing tenant isolation token %q", token)
		}
	}
}
