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

func TestEvaluateClampsConfidenceAndIgnoresInvalidSeverity(t *testing.T) {
	got := Evaluate(Input{
		DetectionRuleCount:       100,
		OrbitalCandidateSeverity: "unknown",
		OrbitalConfidence:        5,
		AgentStatus:              "quarantined",
		RecentAgentAuditEvents:   100,
		ManeuverEvents:           1,
		ConjunctionEvents:        1,
		OrbitalOpsConfidence:     4,
		OrbitalOpsSeverity:       "critical",
	})
	if got.Confidence != 0.99 {
		t.Fatalf("confidence = %.2f, want capped 0.99", got.Confidence)
	}
	if got.CandidateSeverity != "critical" {
		t.Fatalf("candidate severity = %q, want critical", got.CandidateSeverity)
	}
}

func TestEvaluateNegativeConfidenceDoesNotAddWeight(t *testing.T) {
	got := Evaluate(Input{
		OrbitalCandidateSeverity: "medium",
		OrbitalConfidence:        -1,
		ManeuverEvents:           1,
		OrbitalOpsConfidence:     -2,
		OrbitalOpsSeverity:       "high",
	})
	if got.Confidence != 0.35 {
		t.Fatalf("confidence = %.2f, want base 0.35", got.Confidence)
	}
	if got.CandidateSeverity != "high" {
		t.Fatalf("candidate severity = %q, want high", got.CandidateSeverity)
	}
}

func TestHigherSeverityAndUniqueHelpers(t *testing.T) {
	if got := higherSeverity("high", "low"); got != "high" {
		t.Fatalf("higherSeverity(high, low) = %q", got)
	}
	if got := higherSeverity("", "medium"); got != "medium" {
		t.Fatalf("higherSeverity(empty, medium) = %q", got)
	}
	got := unique([]string{"rf", "", "rf", "audit", "audit"})
	if len(got) != 2 || got[0] != "rf" || got[1] != "audit" {
		t.Fatalf("unique = %#v", got)
	}
}
