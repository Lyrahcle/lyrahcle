package sparta

import (
	"strings"
	"testing"
	"time"
)

func TestCatalogIsInternallyConsistent(t *testing.T) {
	for _, token := range []string{"Manually maintained", "curated subset", "published matrix", "reports", "procurement"} {
		if !strings.Contains(CatalogScopeNote, token) {
			t.Fatalf("catalog scope note must contain %q", token)
		}
	}
	if CatalogVersion == "" {
		t.Fatal("the curated catalog must have a version")
	}

	seen := map[string]bool{}
	for _, technique := range Techniques() {
		if seen[technique.ID] {
			t.Fatalf("duplicate technique %s", technique.ID)
		}
		seen[technique.ID] = true
		if _, ok := TacticByID(technique.TacticID); !ok {
			t.Fatalf("technique %s references unknown tactic %s", technique.ID, technique.TacticID)
		}
		if technique.Name == "" || technique.Description == "" || technique.Segment == "" {
			t.Fatalf("technique %s is missing descriptive fields", technique.ID)
		}
	}
	for _, rule := range rules {
		for _, techniqueID := range rule.TechniqueIDs {
			if _, ok := TechniqueByID(techniqueID); !ok {
				t.Fatalf("rule %s/%s maps to unknown technique %s", rule.EvidenceKind, rule.Match, techniqueID)
			}
		}
		if rule.Rationale == "" {
			t.Fatalf("rule %s/%s has no rationale", rule.EvidenceKind, rule.Match)
		}
		if rule.Confidence < 0 || rule.Confidence > 75 {
			t.Fatalf("rule %s/%s has an out-of-range confidence %d", rule.EvidenceKind, rule.Match, rule.Confidence)
		}
	}
}

func TestAssessIsDeterministicAndScoped(t *testing.T) {
	first := Assess(EvidenceTMTCFinding, "sequence_repeat", "frame-1")
	second := Assess(EvidenceTMTCFinding, "SEQUENCE_REPEAT", "frame-1")
	if len(first) == 0 || len(first) != len(second) {
		t.Fatalf("expected matching assessments, got %d and %d", len(first), len(second))
	}
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("assessment %d is not deterministic", index)
		}
	}
	if len(Assess(EvidenceTMTCFinding, "unknown_finding", "frame-1")) != 0 {
		t.Fatal("expected unmapped evidence to produce no assessment")
	}
}

func TestExternalReceptionStaysContextOnly(t *testing.T) {
	assessments := Assess(EvidenceExternalRecep, "community_observation", "satnogs-1")
	if len(assessments) == 0 {
		t.Fatal("expected community reception to map to reconnaissance context")
	}
	for _, assessment := range assessments {
		if !assessment.ContextOnly {
			t.Fatalf("technique %s must stay context only", assessment.TechniqueID)
		}
		if assessment.Confidence > 20 {
			t.Fatalf("technique %s has too much confidence for third-party context", assessment.TechniqueID)
		}
	}
}

func TestAssessAnomalyScalesWithMagnitudeButStaysBounded(t *testing.T) {
	low := AssessAnomaly("power_dbm", 1, "anomaly-1")
	high := AssessAnomaly("power_dbm", 12, "anomaly-2")
	if len(low) == 0 || len(high) != len(low) {
		t.Fatalf("expected equal technique sets, got %d and %d", len(low), len(high))
	}
	if high[0].Confidence <= low[0].Confidence {
		t.Fatalf("expected a stronger anomaly to score higher: %d vs %d", high[0].Confidence, low[0].Confidence)
	}
	if high[0].Confidence > 75 {
		t.Fatalf("confidence must stay bounded, got %d", high[0].Confidence)
	}
	if !high[0].ContextOnly {
		t.Fatal("raw RF anomalies must remain context only until physically validated")
	}
}

func TestBuildMatrixCountsDistinctEvidence(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	assessments := AssessIncident("agent_posture", "incident-1")
	if len(assessments) == 0 {
		t.Fatal("expected agent posture incidents to map")
	}
	observations := []ObservedAssessment{
		{Assessment: assessments[0], ObservedAt: now.Add(-2 * time.Hour), EvidenceRef: "incident-1"},
		{Assessment: assessments[0], ObservedAt: now.Add(-time.Hour), EvidenceRef: "incident-1"},
		{Assessment: assessments[0], ObservedAt: now, EvidenceRef: "incident-2"},
	}
	matrix := BuildMatrix(observations, now)
	if matrix.EvidenceTotal != 2 {
		t.Fatalf("expected duplicate evidence references to collapse, got %d", matrix.EvidenceTotal)
	}
	if matrix.ObservedTotal != 1 {
		t.Fatalf("expected one observed technique, got %d", matrix.ObservedTotal)
	}
	if matrix.TechniqueTotal != len(Techniques()) {
		t.Fatalf("matrix must cover the whole catalog, got %d of %d", matrix.TechniqueTotal, len(Techniques()))
	}
	if matrix.ObservedPercentage < 0 || matrix.ObservedPercentage > 100 {
		t.Fatalf("unexpected coverage percentage %d", matrix.ObservedPercentage)
	}

	found := false
	for _, tactic := range matrix.Tactics {
		for _, technique := range tactic.Techniques {
			if technique.TechniqueID != assessments[0].TechniqueID {
				continue
			}
			found = true
			if technique.EvidenceCount != 2 {
				t.Fatalf("expected 2 evidence items, got %d", technique.EvidenceCount)
			}
			if !technique.LastObservedAt.Equal(now) {
				t.Fatalf("unexpected last observation %v", technique.LastObservedAt)
			}
			if technique.ContextOnly {
				t.Fatal("triaged incident evidence should not be context only")
			}
		}
	}
	if !found {
		t.Fatal("expected the mapped technique to appear in the matrix")
	}
}

func TestBuildMatrixWithoutEvidenceIsEmptyButComplete(t *testing.T) {
	matrix := BuildMatrix(nil, time.Now().UTC())
	if matrix.CatalogVersion != CatalogVersion || matrix.CatalogScopeNote != CatalogScopeNote {
		t.Fatal("every coverage response must carry the curated catalog version and scope note")
	}
	if matrix.ObservedTotal != 0 || matrix.EvidenceTotal != 0 {
		t.Fatalf("expected an empty matrix, got %d observed and %d evidence", matrix.ObservedTotal, matrix.EvidenceTotal)
	}
	if len(matrix.Tactics) != len(Tactics()) {
		t.Fatalf("expected every tactic, got %d", len(matrix.Tactics))
	}
	if matrix.DetectableTotal == 0 || matrix.DetectableTotal > matrix.TechniqueTotal {
		t.Fatalf("unexpected detectable total %d", matrix.DetectableTotal)
	}
}
