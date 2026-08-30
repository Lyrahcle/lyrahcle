package store

import (
	"os"
	"strings"
	"testing"
)

func TestOperationalTenantStoresContainOrganizationPredicates(t *testing.T) {
	for _, file := range []string{"programs.go", "rf.go", "anomalies.go", "orbital_events.go", "search.go", "asset_alerts.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "organization_id") {
			t.Fatalf("%s missing organization scope", file)
		}
	}
}
