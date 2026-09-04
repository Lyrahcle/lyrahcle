package main

import (
	"context"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
)

type automationTestKey struct{ kid string }

func (k automationTestKey) KID() string       { return k.kid }
func (k automationTestKey) Algorithm() string { return "Ed25519" }
func (k automationTestKey) Sign(context.Context, []byte) ([]byte, error) {
	return make([]byte, ed25519.SignatureSize), nil
}

type automationLifecycleProvider struct {
	keys         []keyprovider.KeyMetadata
	policy       keyprovider.RotationPolicy
	prepareCalls int
}

func (p *automationLifecycleProvider) ActiveSigningKey(context.Context) (keyprovider.SigningKey, error) {
	for _, key := range p.keys {
		if key.Status == keyprovider.KeyStatusActive {
			return automationTestKey{kid: key.KID}, nil
		}
	}
	return nil, keyprovider.ErrKeyNotFound
}
func (p *automationLifecycleProvider) SigningKey(_ context.Context, kid string) (keyprovider.SigningKey, error) {
	for _, key := range p.keys {
		if key.KID == kid && (key.Status == keyprovider.KeyStatusActive || key.Status == keyprovider.KeyStatusGrace) {
			return automationTestKey{kid: kid}, nil
		}
	}
	return nil, keyprovider.ErrKeyNotFound
}
func (p *automationLifecycleProvider) Keys(context.Context) ([]keyprovider.KeyMetadata, error) {
	return append([]keyprovider.KeyMetadata(nil), p.keys...), nil
}
func (p *automationLifecycleProvider) PrepareRotation(context.Context) (keyprovider.KeyMetadata, []byte, error) {
	p.prepareCalls++
	for _, key := range p.keys {
		if key.Status == keyprovider.KeyStatusStaged {
			return keyprovider.KeyMetadata{}, nil, keyprovider.ErrRotationPending
		}
	}
	created := time.Now().UTC()
	key := keyprovider.KeyMetadata{KID: "ed25519-sha256-auto-staged", Algorithm: "Ed25519", Status: keyprovider.KeyStatusStaged, CreatedAt: created}
	p.keys = append(p.keys, key)
	return key, []byte("PUBLIC"), nil
}
func (p *automationLifecycleProvider) Activate(context.Context, string, time.Duration) (keyprovider.KeyMetadata, error) {
	return keyprovider.KeyMetadata{}, errors.New("unexpected activate")
}
func (p *automationLifecycleProvider) Revoke(context.Context, string) (keyprovider.KeyMetadata, error) {
	return keyprovider.KeyMetadata{}, errors.New("unexpected revoke")
}
func (p *automationLifecycleProvider) PublicKeyPEM(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (p *automationLifecycleProvider) RotationPolicy(context.Context) (keyprovider.RotationPolicy, error) {
	return p.policy, nil
}
func (p *automationLifecycleProvider) UpdateRotationPolicy(_ context.Context, policy keyprovider.RotationPolicy) (keyprovider.RotationPolicy, error) {
	p.policy = policy
	return policy, nil
}

func TestSigningKeyRotationCheckAutoStagesDueKeyWithoutActivation(t *testing.T) {
	now := time.Date(2026, 8, 30, 20, 0, 0, 0, time.UTC)
	activated := now.Add(-91 * 24 * time.Hour)
	provider := &automationLifecycleProvider{
		keys:   []keyprovider.KeyMetadata{{KID: "ed25519-sha256-active", Algorithm: "Ed25519", Status: keyprovider.KeyStatusActive, CreatedAt: activated, ActivatedAt: &activated}},
		policy: keyprovider.RotationPolicy{AutomaticPrepare: true, IntervalSeconds: int64((90 * 24 * time.Hour) / time.Second), StagedAlertAfterSeconds: 86400, GraceAlertBeforeSeconds: 86400},
	}
	if err := runSigningKeyRotationCheck(context.Background(), provider, nil, now); err != nil {
		t.Fatal(err)
	}
	if provider.prepareCalls != 1 {
		t.Fatalf("prepare calls=%d want 1", provider.prepareCalls)
	}
	active := 0
	staged := 0
	for _, key := range provider.keys {
		if key.Status == keyprovider.KeyStatusActive {
			active++
		}
		if key.Status == keyprovider.KeyStatusStaged {
			staged++
		}
	}
	if active != 1 || staged != 1 {
		t.Fatalf("keys=%+v; automation must stage without activation", provider.keys)
	}
}

func TestSigningKeyRotationCheckDoesNotPrepareWhenDisabledOrPending(t *testing.T) {
	now := time.Date(2026, 8, 30, 20, 0, 0, 0, time.UTC)
	old := now.Add(-100 * 24 * time.Hour)
	base := keyprovider.RotationPolicy{AutomaticPrepare: false, IntervalSeconds: int64((90 * 24 * time.Hour) / time.Second), StagedAlertAfterSeconds: 86400, GraceAlertBeforeSeconds: 86400}
	provider := &automationLifecycleProvider{keys: []keyprovider.KeyMetadata{{KID: "active", Status: keyprovider.KeyStatusActive, CreatedAt: old}}, policy: base}
	if err := runSigningKeyRotationCheck(context.Background(), provider, nil, now); err != nil {
		t.Fatal(err)
	}
	if provider.prepareCalls != 0 {
		t.Fatalf("disabled automation prepared %d keys", provider.prepareCalls)
	}

	provider.policy.AutomaticPrepare = true
	provider.keys = append(provider.keys, keyprovider.KeyMetadata{KID: "staged", Status: keyprovider.KeyStatusStaged, CreatedAt: now.Add(-2 * time.Hour)})
	if err := runSigningKeyRotationCheck(context.Background(), provider, nil, now); err != nil {
		t.Fatal(err)
	}
	if provider.prepareCalls != 0 {
		t.Fatalf("pending rotation prepared another key")
	}
}
