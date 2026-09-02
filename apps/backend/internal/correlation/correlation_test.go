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


func TestEvaluateAssetAlertsNeverRaiseSeverityOrConfidence(t *testing.T) {
	baseline := Evaluate(Input{DetectionRuleCount: 1})
	withAlerts := Evaluate(Input{DetectionRuleCount: 1, ContextAssetAlerts: 50})
	if withAlerts.Confidence != baseline.Confidence {
		t.Fatalf("context-only CVE alerts changed confidence: baseline %.2f, with alerts %.2f", baseline.Confidence, withAlerts.Confidence)
	}
	if withAlerts.CandidateSeverity != baseline.CandidateSeverity {
		t.Fatalf("context-only CVE alerts changed candidate severity: baseline %q, with alerts %q", baseline.CandidateSeverity, withAlerts.CandidateSeverity)
	}
	if len(withAlerts.ContextOnlyDomains) != 1 || withAlerts.ContextOnlyDomains[0] != "asset_alerts" {
		t.Fatalf("context-only domains = %#v, want asset_alerts", withAlerts.ContextOnlyDomains)
	}
}

func TestEvaluateActiveAgentDoesNotCreatePostureFinding(t *testing.T) {
	got := Evaluate(Input{DetectionRuleCount: 1, AgentStatus: "active"})
	if got.CandidateSeverity != "" {
		t.Fatalf("active agent candidate severity = %q, want empty", got.CandidateSeverity)
	}
	for _, domain := range got.CorroboratingDomains {
		if domain == "agent_posture" {
			t.Fatalf("active agent must not add agent_posture: %#v", got.CorroboratingDomains)
		}
	}
}

func TestEvaluateNonActiveAgentPostureIsExplainableCandidate(t *testing.T) {
	for _, status := range []string{"quarantined", "revoked"} {
		t.Run(status, func(t *testing.T) {
			got := Evaluate(Input{DetectionRuleCount: 1, AgentStatus: status})
			if got.CandidateSeverity != "critical" {
				t.Fatalf("candidate severity = %q, want critical", got.CandidateSeverity)
			}
			foundDomain := false
			foundReason := false
			for _, domain := range got.CorroboratingDomains {
				if domain == "agent_posture" {
					foundDomain = true
				}
			}
			for _, reason := range got.Reasons {
				if reason == "agent identity posture is not active" {
					foundReason = true
				}
			}
			if !foundDomain || !foundReason {
				t.Fatalf("posture finding is not explainable: domains=%#v reasons=%#v", got.CorroboratingDomains, got.Reasons)
			}
		})
	}
}

func TestEvaluateZeroRuleInputRemainsBounded(t *testing.T) {
	got := Evaluate(Input{})
	if got.Confidence != 0.35 {
		t.Fatalf("confidence = %.2f, want 0.35", got.Confidence)
	}
	if got.CandidateSeverity != "" {
		t.Fatalf("candidate severity = %q, want empty", got.CandidateSeverity)
	}
	if len(got.CorroboratingDomains) != 1 || got.CorroboratingDomains[0] != "rf" {
		t.Fatalf("domains = %#v, want rf baseline", got.CorroboratingDomains)
	}
}
