package keyprovider

import (
	"context"
	"errors"
	"sort"
	"time"
)

const (
	minRotationInterval = 24 * time.Hour
	maxRotationInterval = 365 * 24 * time.Hour
	minAlertWindow      = time.Hour
	maxAlertWindow      = 30 * 24 * time.Hour
)

type RotationPolicy struct {
	AutomaticPrepare        bool  `json:"automatic_prepare"`
	IntervalSeconds         int64 `json:"interval_seconds"`
	StagedAlertAfterSeconds int64 `json:"staged_alert_after_seconds"`
	GraceAlertBeforeSeconds int64 `json:"grace_alert_before_seconds"`
}

func DefaultRotationPolicy() RotationPolicy {
	return RotationPolicy{
		AutomaticPrepare:        false,
		IntervalSeconds:         int64((90 * 24 * time.Hour) / time.Second),
		StagedAlertAfterSeconds: int64((24 * time.Hour) / time.Second),
		GraceAlertBeforeSeconds: int64((24 * time.Hour) / time.Second),
	}
}

func NormalizeRotationPolicy(policy RotationPolicy) (RotationPolicy, bool) {
	defaults := DefaultRotationPolicy()
	changed := false
	if policy.IntervalSeconds == 0 {
		policy.IntervalSeconds = defaults.IntervalSeconds
		changed = true
	}
	if policy.StagedAlertAfterSeconds == 0 {
		policy.StagedAlertAfterSeconds = defaults.StagedAlertAfterSeconds
		changed = true
	}
	if policy.GraceAlertBeforeSeconds == 0 {
		policy.GraceAlertBeforeSeconds = defaults.GraceAlertBeforeSeconds
		changed = true
	}
	return policy, changed
}

func ValidateRotationPolicy(policy RotationPolicy) error {
	interval := time.Duration(policy.IntervalSeconds) * time.Second
	staged := time.Duration(policy.StagedAlertAfterSeconds) * time.Second
	grace := time.Duration(policy.GraceAlertBeforeSeconds) * time.Second
	if interval < minRotationInterval || interval > maxRotationInterval {
		return errors.New("keyprovider: rotation interval must be between 24h and 8760h")
	}
	if staged < minAlertWindow || staged > maxAlertWindow {
		return errors.New("keyprovider: staged alert window must be between 1h and 720h")
	}
	if grace < minAlertWindow || grace > maxAlertWindow {
		return errors.New("keyprovider: grace alert window must be between 1h and 720h")
	}
	return nil
}

type RotationWarning struct {
	Code     string     `json:"code"`
	Severity string     `json:"severity"`
	KID      string     `json:"kid,omitempty"`
	Deadline *time.Time `json:"deadline,omitempty"`
}

type RotationHealth struct {
	RotationDue    bool              `json:"rotation_due"`
	NextRotationAt *time.Time        `json:"next_rotation_at,omitempty"`
	StagedPending  bool              `json:"staged_pending"`
	Warnings       []RotationWarning `json:"warnings"`
}

func AssessRotation(keys []KeyMetadata, policy RotationPolicy, now time.Time) RotationHealth {
	normalized, _ := NormalizeRotationPolicy(policy)
	health := RotationHealth{Warnings: []RotationWarning{}}
	interval := time.Duration(normalized.IntervalSeconds) * time.Second
	stagedAlert := time.Duration(normalized.StagedAlertAfterSeconds) * time.Second
	graceAlert := time.Duration(normalized.GraceAlertBeforeSeconds) * time.Second
	var active *KeyMetadata
	for i := range keys {
		key := keys[i]
		switch key.Status {
		case KeyStatusActive:
			if active == nil || key.CreatedAt.After(active.CreatedAt) {
				copy := key
				active = &copy
			}
		case KeyStatusStaged:
			health.StagedPending = true
			if !key.CreatedAt.IsZero() && !now.Before(key.CreatedAt.Add(stagedAlert)) {
				health.Warnings = append(health.Warnings, RotationWarning{Code: "staged_pending", Severity: "warning", KID: key.KID})
			}
		case KeyStatusGrace:
			if key.GraceExpiresAt != nil && !now.Before(key.GraceExpiresAt.Add(-graceAlert)) {
				deadline := key.GraceExpiresAt.UTC()
				health.Warnings = append(health.Warnings, RotationWarning{Code: "grace_expiring", Severity: "warning", KID: key.KID, Deadline: &deadline})
			}
		}
	}
	if active == nil {
		health.Warnings = append(health.Warnings, RotationWarning{Code: "no_active_key", Severity: "critical"})
	} else if interval > 0 && !active.CreatedAt.IsZero() {
		rotationBase := active.CreatedAt
		if active.ActivatedAt != nil && !active.ActivatedAt.IsZero() {
			rotationBase = active.ActivatedAt.UTC()
		}
		next := rotationBase.Add(interval).UTC()
		health.NextRotationAt = &next
		if !now.Before(next) {
			health.RotationDue = true
			health.Warnings = append(health.Warnings, RotationWarning{Code: "rotation_due", Severity: "warning", KID: active.KID, Deadline: &next})
		}
	}
	sort.SliceStable(health.Warnings, func(i, j int) bool {
		if health.Warnings[i].Severity != health.Warnings[j].Severity {
			return health.Warnings[i].Severity == "critical"
		}
		return health.Warnings[i].Code < health.Warnings[j].Code
	})
	return health
}

func (p *ManagedLocalProvider) RotationPolicy(context.Context) (RotationPolicy, error) {
	if p == nil {
		return RotationPolicy{}, ErrLifecycleUnsupported
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	policy, _ := NormalizeRotationPolicy(p.manifest.RotationPolicy)
	return policy, nil
}

func (p *ManagedLocalProvider) UpdateRotationPolicy(_ context.Context, policy RotationPolicy) (RotationPolicy, error) {
	if p == nil {
		return RotationPolicy{}, ErrLifecycleUnsupported
	}
	policy, _ = NormalizeRotationPolicy(policy)
	if err := ValidateRotationPolicy(policy); err != nil {
		return RotationPolicy{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.manifest.RotationPolicy = policy
	if err := p.saveManifestLocked(); err != nil {
		return RotationPolicy{}, err
	}
	return policy, nil
}
