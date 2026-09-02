package tlequality

import (
	"testing"
	"time"
)

func TestEpoch(t *testing.T) {
	epoch, err := Epoch("1 25544U 98067A   26239.50000000  .00008717  00000+0  16243-3 0  9995")
	if err != nil {
		t.Fatalf("Epoch returned error: %v", err)
	}
	want := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	if !epoch.Equal(want) {
		t.Fatalf("epoch = %s, want %s", epoch, want)
	}
}

func TestEpochRejectsInvalidData(t *testing.T) {
	cases := []string{
		"short",
		"1 25544U 98067A   XX239.50000000  .00008717  00000+0  16243-3 0  9995",
		"1 25544U 98067A   26367.00000000  .00008717  00000+0  16243-3 0  9995",
	}
	for _, line := range cases {
		if _, err := Epoch(line); err == nil {
			t.Fatalf("Epoch(%q) should fail", line)
		}
	}
}

func TestAssess(t *testing.T) {
	policy := Policy{CorrelationMaxAge: 24 * time.Hour, StaleMaxAge: 72 * time.Hour}
	now := time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)

	fresh := now.Add(-12 * time.Hour)
	assessment := Assess(&fresh, now, policy)
	if assessment.Status != StatusFresh || !assessment.EscalationAllowed || assessment.Confidence != "high" {
		t.Fatalf("fresh assessment = %+v", assessment)
	}

	aging := now.Add(-48 * time.Hour)
	assessment = Assess(&aging, now, policy)
	if assessment.Status != StatusAging || assessment.EscalationAllowed || assessment.Confidence != "degraded" {
		t.Fatalf("aging assessment = %+v", assessment)
	}

	stale := now.Add(-96 * time.Hour)
	assessment = Assess(&stale, now, policy)
	if assessment.Status != StatusStale || assessment.EscalationAllowed || assessment.Confidence != "unavailable" {
		t.Fatalf("stale assessment = %+v", assessment)
	}

	assessment = Assess(nil, now, policy)
	if assessment.Status != StatusUnusable || assessment.EscalationAllowed {
		t.Fatalf("unusable assessment = %+v", assessment)
	}
}

func TestAssessRejectsFarFutureEpoch(t *testing.T) {
	policy := Policy{CorrelationMaxAge: 24 * time.Hour, StaleMaxAge: 72 * time.Hour}
	now := time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)
	future := now.Add(12 * time.Hour)
	assessment := Assess(&future, now, policy)
	if assessment.Status != StatusUnusable {
		t.Fatalf("future assessment = %+v", assessment)
	}
}


func TestAssessFailsClosedForInvalidPolicy(t *testing.T) {
	now := time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)
	epoch := now.Add(-time.Hour)
	for _, policy := range []Policy{
		{},
		{CorrelationMaxAge: 24 * time.Hour, StaleMaxAge: 24 * time.Hour},
		{CorrelationMaxAge: -time.Hour, StaleMaxAge: 72 * time.Hour},
	} {
		assessment := Assess(&epoch, now, policy)
		if assessment.Status != StatusUnusable || assessment.EscalationAllowed {
			t.Fatalf("invalid policy assessment = %+v, want unusable/fail-closed", assessment)
		}
	}
}
