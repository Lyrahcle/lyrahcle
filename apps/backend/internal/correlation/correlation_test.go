package correlation

import "testing"

func TestEvaluateSeparatesContextOnlySignals(t *testing.T) {
	got := Evaluate(Input{
		DetectionRuleCount:       2,
		OrbitalCandidateSeverity: "high",
		OrbitalConfidence:        0.8,
		AgentStatus:              "active",
		RecentAgentAuditEvents:   2,
		ContextAssetAlerts:       3,
	})
	if got.CandidateSeverity != "high" {
		t.Fatalf("candidate severity = %q, want high", got.CandidateSeverity)
	}
	if got.Confidence <= 0.5 || got.Confidence > 0.99 {
		t.Fatalf("unexpected confidence %.2f", got.Confidence)
	}
	if len(got.ContextOnlyDomains) != 1 || got.ContextOnlyDomains[0] != "asset_alerts" {
		t.Fatalf("context-only domains = %#v", got.ContextOnlyDomains)
	}
}

func TestEvaluateAgentPostureCanRaiseCandidateOnly(t *testing.T) {
	got := Evaluate(Input{DetectionRuleCount: 1, AgentStatus: "quarantined"})
	if got.CandidateSeverity != "critical" {
		t.Fatalf("candidate severity = %q, want critical", got.CandidateSeverity)
	}
	found := false
	for _, d := range got.CorroboratingDomains {
		if d == "agent_posture" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing agent_posture domain: %#v", got.CorroboratingDomains)
	}
}

func TestEvaluateOrbitalOperationsContext(t *testing.T) {
	result := Evaluate(Input{
		DetectionRuleCount: 1, ManeuverEvents: 1, ConjunctionEvents: 1,
		OrbitalOpsConfidence: 0.9, OrbitalOpsSeverity: "high",
	})
	if result.CandidateSeverity != "high" {
		t.Fatalf("candidate severity = %q, want high", result.CandidateSeverity)
	}
	found := false
	for _, domain := range result.CorroboratingDomains {
		if domain == "orbital_operations" {
			found = true
		}
	}
	if !found {
		t.Fatal("orbital_operations domain missing")
	}
}
