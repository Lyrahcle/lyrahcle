package main

import (
	"os"
	"strings"
	"testing"
)

func TestFleetRolloutHardeningContractV1465(t *testing.T) {
	handlerBytes, err := os.ReadFile("fleet_rollouts_v141.go")
	if err != nil {
		t.Fatal(err)
	}
	handler := string(handlerBytes)
	for _, token := range []string{
		`AckTimeoutSeconds       int`,
		`json:"ack_timeout_seconds"`,
		"rollouts.EvaluateTargets",
		"rollouts.OverdueIssuedCount",
		"rollout acknowledgement timeout exceeded",
		"rollout target policy changed",
		"audit.ActionAgentRolloutFailed",
		"http.StatusUnprocessableEntity",
	} {
		if !strings.Contains(handler, token) {
			t.Fatalf("fleet rollout hardening handler missing %q", token)
		}
	}

	migrationBytes, err := os.ReadFile("../../migrations/0042_fleet_rollout_hardening.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(migrationBytes)
	for _, token := range []string{"ack_timeout_seconds", "BETWEEN 30 AND 86400", "agent_fleet_targets_issued_deadline_idx", "WHERE status = 'issued'"} {
		if !strings.Contains(migration, token) {
			t.Fatalf("fleet rollout hardening migration missing %q", token)
		}
	}

	auditBytes, err := os.ReadFile("../../internal/audit/audit.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(auditBytes), `ActionAgentRolloutFailed         = "agent.rollout_failed"`) {
		t.Fatal("central audit catalog missing agent.rollout_failed")
	}
}
