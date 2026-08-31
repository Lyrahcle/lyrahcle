package keyprovider

import (
	"context"
	"testing"
	"time"
)

func TestManagedRotationPolicyPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	provider, err := NewManagedLocalProvider(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	policy := RotationPolicy{
		AutomaticPrepare:        true,
		IntervalSeconds:         int64((45 * 24 * time.Hour) / time.Second),
		StagedAlertAfterSeconds: int64((12 * time.Hour) / time.Second),
		GraceAlertBeforeSeconds: int64((6 * time.Hour) / time.Second),
	}
	if _, err := provider.UpdateRotationPolicy(context.Background(), policy); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewManagedLocalProvider(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := reloaded.RotationPolicy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != policy {
		t.Fatalf("policy after restart = %#v want %#v", got, policy)
	}
}

func TestAssessRotationWarnsForDueStagedAndGrace(t *testing.T) {
	now := time.Date(2026, 8, 30, 20, 0, 0, 0, time.UTC)
	graceDeadline := now.Add(30 * time.Minute)
	keys := []KeyMetadata{
		{KID: "ed25519-sha256-active", Status: KeyStatusActive, CreatedAt: now.Add(-91 * 24 * time.Hour)},
		{KID: "ed25519-sha256-staged", Status: KeyStatusStaged, CreatedAt: now.Add(-25 * time.Hour)},
		{KID: "ed25519-sha256-grace", Status: KeyStatusGrace, CreatedAt: now.Add(-100 * 24 * time.Hour), GraceExpiresAt: &graceDeadline},
	}
	health := AssessRotation(keys, DefaultRotationPolicy(), now)
	if !health.RotationDue || !health.StagedPending {
		t.Fatalf("health=%+v", health)
	}
	codes := map[string]bool{}
	for _, warning := range health.Warnings {
		codes[warning.Code] = true
	}
	for _, code := range []string{"rotation_due", "staged_pending", "grace_expiring"} {
		if !codes[code] {
			t.Fatalf("missing warning %q in %+v", code, health.Warnings)
		}
	}
}

func TestValidateRotationPolicyRejectsUnsafeWindows(t *testing.T) {
	policy := DefaultRotationPolicy()
	policy.IntervalSeconds = 3600
	if err := ValidateRotationPolicy(policy); err == nil {
		t.Fatal("expected one-hour rotation interval to be rejected")
	}
	policy = DefaultRotationPolicy()
	policy.StagedAlertAfterSeconds = 60
	if err := ValidateRotationPolicy(policy); err == nil {
		t.Fatal("expected one-minute staged alert window to be rejected")
	}
}

func TestAssessRotationUsesActivationTimeWhenAvailable(t *testing.T) {
	now := time.Date(2026, 8, 30, 20, 0, 0, 0, time.UTC)
	activated := now.Add(-10 * 24 * time.Hour)
	keys := []KeyMetadata{{
		KID:         "ed25519-sha256-active",
		Status:      KeyStatusActive,
		CreatedAt:   now.Add(-120 * 24 * time.Hour),
		ActivatedAt: &activated,
	}}
	health := AssessRotation(keys, DefaultRotationPolicy(), now)
	if health.RotationDue {
		t.Fatalf("rotation should be based on activation time, health=%+v", health)
	}
	if health.NextRotationAt == nil {
		t.Fatal("expected next rotation time")
	}
	want := activated.Add(90 * 24 * time.Hour)
	if !health.NextRotationAt.Equal(want) {
		t.Fatalf("next rotation=%s want %s", health.NextRotationAt, want)
	}
}

func TestNormalizeRotationPolicyRestoresSafeDefaults(t *testing.T) {
	got, changed := NormalizeRotationPolicy(RotationPolicy{})
	if !changed {
		t.Fatal("expected empty policy to be normalized")
	}
	if got != DefaultRotationPolicy() {
		t.Fatalf("normalized=%+v want=%+v", got, DefaultRotationPolicy())
	}
}

func TestUpdateRotationPolicyRejectsInvalidPolicy(t *testing.T) {
	provider, err := NewManagedLocalProvider(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultRotationPolicy()
	policy.GraceAlertBeforeSeconds = 60
	if _, err := provider.UpdateRotationPolicy(context.Background(), policy); err == nil {
		t.Fatal("expected unsafe grace alert window to be rejected")
	}
}
