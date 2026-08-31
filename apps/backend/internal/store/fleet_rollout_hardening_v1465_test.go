package store

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestFleetTargetEligibilityAndAckDeadlineV1465(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	orgID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO organizations (id,slug,name) VALUES ($1,$2,$3)`, orgID, "fleet-hardening-"+orgID.String()[:8], "Fleet Hardening"); err != nil {
		t.Fatal(err)
	}

	agents := []string{"fleet-good", "fleet-missing", "fleet-stale", "fleet-changed"}
	for _, agentID := range agents {
		if _, err := pool.Exec(ctx, `INSERT INTO agent_identities (agent_id,organization_id,status) VALUES ($1,$2,'active')`, agentID, orgID); err != nil {
			t.Fatal(err)
		}
	}

	identityStore := NewAgentIdentityStore(pool)
	base := AgentDeviceInfo{
		Driver: "rtlsdr", Hardware: "RTL-SDR Blog V4", RX: true,
		Capabilities: AgentDeviceCapabilities{
			Architecture: "arm64", OS: "linux", DriverVersion: "0.6.0",
			FrequencyMinHz: 24_000_000, FrequencyMaxHz: 1_766_000_000,
			SampleRatesHz: []int64{2_048_000}, Channels: 1,
		},
	}
	if err := identityStore.ReportDeviceInOrganization(ctx, orgID, "fleet-good", base); err != nil {
		t.Fatalf("report good device: %v", err)
	}
	if err := identityStore.ReportDeviceInOrganization(ctx, orgID, "fleet-stale", base); err != nil {
		t.Fatalf("report stale device: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE agent_identities SET device_reported_at=now()-interval '16 minutes' WHERE organization_id=$1 AND agent_id='fleet-stale'`, orgID); err != nil {
		t.Fatal(err)
	}

	changed := base
	changed.Capabilities.IntegrityManifestSHA256 = strings.Repeat("a", 64)
	if err := identityStore.ReportDeviceInOrganization(ctx, orgID, "fleet-changed", changed); err != nil {
		t.Fatalf("report integrity baseline: %v", err)
	}
	changed.Capabilities.IntegrityManifestSHA256 = strings.Repeat("b", 64)
	if err := identityStore.ReportDeviceInOrganization(ctx, orgID, "fleet-changed", changed); err != nil {
		t.Fatalf("report changed integrity: %v", err)
	}

	fleet := NewFleetRolloutStore(pool)
	eligibility, err := fleet.EvaluateTargets(ctx, orgID, agents, 437_500_000)
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]FleetTargetEligibility, len(eligibility))
	for _, item := range eligibility {
		byID[item.AgentID] = item
	}
	if !byID["fleet-good"].Eligible {
		t.Fatalf("good target rejected: %+v", byID["fleet-good"])
	}
	if byID["fleet-missing"].Eligible || !strings.Contains(byID["fleet-missing"].Reason, "hardware capability report") {
		t.Fatalf("missing hardware target = %+v", byID["fleet-missing"])
	}
	if byID["fleet-stale"].Eligible || byID["fleet-stale"].Compatibility != "stale" {
		t.Fatalf("stale target = %+v", byID["fleet-stale"])
	}
	if byID["fleet-changed"].Eligible || !strings.Contains(byID["fleet-changed"].Reason, "integrity manifest changed") {
		t.Fatalf("changed integrity target = %+v", byID["fleet-changed"])
	}

	outside, err := fleet.EvaluateTargets(ctx, orgID, []string{"fleet-good"}, 2_000_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(outside) != 1 || outside[0].Eligible || !strings.Contains(outside[0].Reason, "outside reported hardware range") {
		t.Fatalf("out-of-range target = %+v", outside)
	}

	rollout, err := fleet.Create(ctx, orgID, "ack deadline", AgentConfigSettings{
		FrequencyHz: 437_500_000, CaptureIntervalSeconds: 10,
	}, 1, 20, 0, 30, []string{"fleet-good"}, nil)
	if err != nil {
		t.Fatalf("create rollout: %v", err)
	}
	if rollout.AckTimeoutSeconds != 30 {
		t.Fatalf("ack timeout = %d, want 30", rollout.AckTimeoutSeconds)
	}
	if err := fleet.MarkIssued(ctx, orgID, rollout.ID, "fleet-good", nil, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE agent_fleet_rollout_targets SET issued_at=now()-interval '31 seconds' WHERE rollout_id=$1 AND organization_id=$2 AND agent_id='fleet-good'`, rollout.ID, orgID); err != nil {
		t.Fatal(err)
	}
	overdue, err := fleet.OverdueIssuedCount(ctx, orgID, rollout.ID, rollout.AckTimeoutSeconds)
	if err != nil {
		t.Fatal(err)
	}
	if overdue != 1 {
		t.Fatalf("overdue issued targets = %d, want 1", overdue)
	}

	var indexExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='agent_fleet_targets_issued_deadline_idx')`).Scan(&indexExists); err != nil {
		t.Fatal(err)
	}
	if !indexExists {
		t.Fatal("ack deadline partial index was not migrated")
	}
}
