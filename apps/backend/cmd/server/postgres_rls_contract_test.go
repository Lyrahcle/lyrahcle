package main

import (
	"os"
	"strings"
	"testing"
)

func TestPostgresRLSContractV142(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/0040_postgres_row_level_security.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(migration)
	required := []string{
		"FORCE ROW LEVEL SECURITY",
		"lyrahcle_rls_allows_organization",
		"current_setting('lyrahcle.rls_request', true)",
		"current_setting('lyrahcle.organization_id', true)",
		"WITH CHECK",
		"agent_fleet_rollouts",
		"rf_detection_rules",
	}
	for _, token := range required {
		if !strings.Contains(sql, token) {
			t.Fatalf("RLS migration missing %q", token)
		}
	}

	serverSource := readServerProductionSource(t)
	if !strings.Contains(serverSource, "r.Use(dbscope.Middleware)") {
		t.Fatal("HTTP router must mark request-originated database work")
	}
	if !strings.Contains(serverSource, "dbscope.WithOrganization(r.Context(), organizationID)") {
		t.Fatal("agent RF ingestion must attach its resolved organization before tenant-scoped queries")
	}

	authSource, err := os.ReadFile("../../internal/auth/middleware.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(authSource), "dbscope.WithOrganization(ctx, claims.OrganizationID)") {
		t.Fatal("authenticated user middleware must propagate organization scope to the database")
	}

	dbSource, err := os.ReadFile("../../internal/db/db.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"BeforeAcquire", "AfterRelease", "lyrahcle.rls_request", "lyrahcle.organization_id", "RESET lyrahcle.rls_request"} {
		if !strings.Contains(string(dbSource), token) {
			t.Fatalf("database pool RLS lifecycle missing %q", token)
		}
	}
}
