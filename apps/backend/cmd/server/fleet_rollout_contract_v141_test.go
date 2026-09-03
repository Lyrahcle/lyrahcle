package main

import (
	"os"
	"strings"
	"testing"
)

func TestFleetRolloutContractV141(t *testing.T) {
	handlerBytes, err := os.ReadFile("fleet_rollouts.go")
	if err != nil {
		t.Fatal(err)
	}
	handler := string(handlerBytes)
	for _, token := range []string{
		`r.With(read).Get("/agent-rollouts"`,
		`r.With(manage).Post("/agent-rollouts"`,
		`/start`, `/advance`, `/pause`, `/resume`, `/rollback`,
		`rollout soak period has not elapsed`, `failure threshold exceeded`,
	} {
		if !strings.Contains(handler, token) {
			t.Fatalf("fleet rollout handler missing %q", token)
		}
	}

	migrationBytes, err := os.ReadFile("../../migrations/0039_agent_fleet_rollouts.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(migrationBytes)
	for _, token := range []string{"agent_fleet_rollouts", "agent_fleet_rollout_targets", "failure_threshold_percent", "soak_seconds", "rolled_back"} {
		if !strings.Contains(migration, token) {
			t.Fatalf("fleet rollout migration missing %q", token)
		}
	}
}
