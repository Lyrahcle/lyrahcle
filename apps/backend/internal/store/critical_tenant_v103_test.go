package store

import (
	"os"
	"strings"
	"testing"
)

func TestCriticalStoresContainOrganizationPredicates(t *testing.T) {
	for _, file := range []string{"assets.go", "audit.go", "incidents.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "organization_id") {
			t.Fatalf("%s does not contain organization scope", file)
		}
	}
}
