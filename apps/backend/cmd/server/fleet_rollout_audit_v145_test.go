package main

import (
	"os"
	"strings"
	"testing"
)

func TestFleetRolloutAdministrativeTransitionsAreAudited(t *testing.T) {
	handlerBytes, err := os.ReadFile("fleet_rollouts_v141.go")
	if err != nil {
		t.Fatal(err)
	}
	handler := string(handlerBytes)

	for _, token := range []string{
		"audit.ActionAgentRolloutCreated",
		"audit.ActionAgentRolloutStarted",
		"audit.ActionAgentRolloutAdvanced",
		"audit.ActionAgentRolloutPaused",
		"audit.ActionAgentRolloutResumed",
		"audit.ActionAgentRolloutRolledBack",
		`map[string]interface{}{"rollout_id":`,
	} {
		if !strings.Contains(handler, token) {
			t.Fatalf("fleet rollout administrative audit contract missing %q", token)
		}
	}

	for _, raw := range []string{
		`"agent.rollout_created"`,
		`"agent.rollout_started"`,
		`"agent.rollout_advanced"`,
		`"agent.rollout_paused"`,
		`"agent.rollout_resumed"`,
		`"agent.rollout_rolled_back"`,
	} {
		if strings.Contains(handler, raw) {
			t.Fatalf("fleet rollout handler must use centralized audit constants instead of raw event %s", raw)
		}
	}
}

func TestFleetRolloutAuditActionsAreCentralized(t *testing.T) {
	auditBytes, err := os.ReadFile("../../internal/audit/audit.go")
	if err != nil {
		t.Fatal(err)
	}
	auditSource := string(auditBytes)

	for name, value := range map[string]string{
		"ActionAgentRolloutCreated":    "agent.rollout_created",
		"ActionAgentRolloutStarted":    "agent.rollout_started",
		"ActionAgentRolloutAdvanced":   "agent.rollout_advanced",
		"ActionAgentRolloutPaused":     "agent.rollout_paused",
		"ActionAgentRolloutResumed":    "agent.rollout_resumed",
		"ActionAgentRolloutRolledBack": "agent.rollout_rolled_back",
	} {
		if !strings.Contains(auditSource, name) || !strings.Contains(auditSource, `"`+value+`"`) {
			t.Fatalf("central audit action catalog missing %s=%q", name, value)
		}
	}
}
