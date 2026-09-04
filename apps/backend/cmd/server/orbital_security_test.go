package main

import (
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/tlequality"
)

func TestSuppressionReason(t *testing.T) {
	cases := []struct {
		name       string
		assessment tlequality.Assessment
		auto       bool
		want       string
	}{
		{name: "aging", assessment: tlequality.Assessment{Status: tlequality.StatusAging}, auto: true, want: "tle_aging"},
		{name: "stale", assessment: tlequality.Assessment{Status: tlequality.StatusStale}, auto: true, want: "tle_stale"},
		{name: "unusable", assessment: tlequality.Assessment{Status: tlequality.StatusUnusable}, auto: true, want: "tle_unusable"},
		{name: "validation mode", assessment: tlequality.Assessment{Status: tlequality.StatusFresh, EscalationAllowed: true}, auto: false, want: "validation_mode"},
		{name: "enabled and fresh", assessment: tlequality.Assessment{Status: tlequality.StatusFresh, EscalationAllowed: true}, auto: true, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := suppressionReason(tc.assessment, tc.auto); got != tc.want {
				t.Fatalf("suppressionReason() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTLEPolicyDefaultsAllowFreshOnly(t *testing.T) {
	policy := tlequality.Policy{CorrelationMaxAge: 24 * time.Hour, StaleMaxAge: 72 * time.Hour}
	now := time.Now().UTC()
	fresh := now.Add(-time.Hour)
	aging := now.Add(-48 * time.Hour)
	if !tlequality.Assess(&fresh, now, policy).EscalationAllowed {
		t.Fatal("fresh TLE should satisfy data quality gate")
	}
	if tlequality.Assess(&aging, now, policy).EscalationAllowed {
		t.Fatal("aging TLE should not satisfy data quality gate")
	}
}
