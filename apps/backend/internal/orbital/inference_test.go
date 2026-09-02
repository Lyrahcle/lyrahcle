package orbital

import (
	"testing"
	"time"
)

func TestInferManeuverCandidate(t *testing.T) {
	t0 := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	prev := StateVector{ObservedAt: t0, XKm: 7000, VYKmS: 7.5}
	obs := StateVector{ObservedAt: t0.Add(time.Second), XKm: 7000, YKm: 9.0, VYKmS: 7.503}
	got := InferManeuverCandidate(prev, obs)
	if !got.Candidate {
		t.Fatalf("expected maneuver candidate: %+v", got)
	}
	if got.DeltaVMS < 2 {
		t.Fatalf("delta-v too small: %v", got.DeltaVMS)
	}
}

func TestInferManeuverRejectsNominal(t *testing.T) {
	t0 := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	prev := StateVector{ObservedAt: t0, XKm: 7000, VYKmS: 7.5}
	obs := StateVector{ObservedAt: t0.Add(time.Second), XKm: 7000, YKm: 7.5, VYKmS: 7.5}
	if InferManeuverCandidate(prev, obs).Candidate {
		t.Fatal("nominal propagation classified as maneuver")
	}
}

func TestAssessConjunctionCandidate(t *testing.T) {
	t0 := time.Now()
	a := StateVector{ObservedAt: t0, XKm: 7000, VYKmS: 7.5}
	b := StateVector{ObservedAt: t0, XKm: 7005, VYKmS: 7.4}
	got := AssessConjunctionCandidate(a, b, 10)
	if !got.Candidate {
		t.Fatalf("expected conjunction candidate: %+v", got)
	}
}
