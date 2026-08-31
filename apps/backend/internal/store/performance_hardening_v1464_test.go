package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPerformanceHardeningContractV1464(t *testing.T) {
	migrationBytes, err := os.ReadFile("../../migrations/0041_performance_hot_paths.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(migrationBytes)
	for _, token := range []string{
		"rf_readings_org_agent_captured_idx",
		"audit_log_actor_occurred_idx",
		"incidents_org_agent_created_idx",
		"incidents_org_assigned_created_idx",
		"incidents_org_satellite_created_idx",
	} {
		if !strings.Contains(migration, token) {
			t.Fatalf("performance migration missing %q", token)
		}
	}

	fleetBytes, err := os.ReadFile("fleet_rollouts.go")
	if err != nil {
		t.Fatal(err)
	}
	fleet := string(fleetBytes)
	for _, token := range []string{
		"unnest($3::text[]) WITH ORDINALITY",
		"tag.RowsAffected() != int64(len(agentIDs))",
		"a.organization_id=$2",
	} {
		if !strings.Contains(fleet, token) {
			t.Fatalf("fleet hot-path optimization missing %q", token)
		}
	}
	if strings.Contains(fleet, "for i, agentID := range agentIDs") {
		t.Fatal("fleet rollout target creation regressed to one INSERT per agent")
	}

	tenantAuditBytes, err := os.ReadFile("tenant_audit.go")
	if err != nil {
		t.Fatal(err)
	}
	tenantAudit := string(tenantAuditBytes)
	if strings.Count(tenantAudit, "s.pool.QueryRow(ctx") != 1 {
		t.Fatalf("tenant snapshot should use one database round trip, source has %d QueryRow calls", strings.Count(tenantAudit, "s.pool.QueryRow(ctx"))
	}
	if strings.Contains(tenantAudit, "for _, q := range queries") {
		t.Fatal("tenant snapshot regressed to sequential count queries")
	}
}

func TestFleetRolloutBulkTargetsV1464(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	orgID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO organizations (id,slug,name) VALUES ($1,$2,$3)`, orgID, "perf-"+orgID.String()[:8], "Performance Test"); err != nil {
		t.Fatal(err)
	}

	agents := []string{"perf-agent-3", "perf-agent-1", "perf-agent-2"}
	for _, agentID := range agents {
		if _, err := pool.Exec(ctx, `INSERT INTO agent_identities (agent_id,organization_id,status) VALUES ($1,$2,'active')`, agentID, orgID); err != nil {
			t.Fatal(err)
		}
	}

	fleet := NewFleetRolloutStore(pool)
	rollout, err := fleet.Create(ctx, orgID, "bulk rollout", AgentConfigSettings{
		FrequencyHz:            437000000,
		CaptureIntervalSeconds: 15,
	}, 2, 20, 0, 900, agents, nil)
	if err != nil {
		t.Fatalf("bulk rollout create: %v", err)
	}
	if len(rollout.Targets) != len(agents) {
		t.Fatalf("target count = %d, want %d", len(rollout.Targets), len(agents))
	}
	for i, target := range rollout.Targets {
		if target.AgentID != agents[i] || target.Ordinal != i {
			t.Fatalf("target %d = (%q,%d), want (%q,%d)", i, target.AgentID, target.Ordinal, agents[i], i)
		}
	}

	_, err = fleet.Create(ctx, orgID, "rollback on missing target", AgentConfigSettings{
		FrequencyHz:            437100000,
		CaptureIntervalSeconds: 20,
	}, 2, 20, 0, 900, []string{agents[0], "missing-agent"}, nil)
	if !errors.Is(err, ErrAgentIdentityNotFound) {
		t.Fatalf("missing target error = %v, want %v", err, ErrAgentIdentityNotFound)
	}
	var leaked int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM agent_fleet_rollouts WHERE organization_id=$1 AND name='rollback on missing target'`, orgID).Scan(&leaked); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Fatalf("failed bulk create leaked %d rollout rows", leaked)
	}

	for _, indexName := range []string{
		"rf_readings_org_agent_captured_idx",
		"audit_log_actor_occurred_idx",
		"incidents_org_agent_created_idx",
		"incidents_org_assigned_created_idx",
		"incidents_org_satellite_created_idx",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname=$1)`, indexName).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("expected migrated index %q", indexName)
		}
	}
}
