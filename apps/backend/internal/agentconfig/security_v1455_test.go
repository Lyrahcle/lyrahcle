package agentconfig

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
)

type fakeSigningKey struct {
	kid, algorithm string
	signature      []byte
	err            error
}

func (k fakeSigningKey) KID() string                                  { return k.kid }
func (k fakeSigningKey) Algorithm() string                            { return k.algorithm }
func (k fakeSigningKey) Sign(context.Context, []byte) ([]byte, error) { return k.signature, k.err }

type fakeProvider struct {
	key keyprovider.SigningKey
	err error
}

func (p fakeProvider) ActiveSigningKey(context.Context) (keyprovider.SigningKey, error) {
	return p.key, p.err
}
func (p fakeProvider) SigningKey(context.Context, string) (keyprovider.SigningKey, error) {
	return p.key, p.err
}

func validFakeKey(t *testing.T) fakeSigningKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := keyprovider.Ed25519KID(pub)
	if err != nil {
		t.Fatal(err)
	}
	return fakeSigningKey{kid: kid, algorithm: "Ed25519", signature: ed25519.Sign(priv, []byte("placeholder"))}
}

func TestNewSignerRequiresProvider(t *testing.T) {
	if _, err := NewSigner(nil); err == nil {
		t.Fatal("nil provider must fail")
	}
}

func TestSignerFailsClosedOnProviderAndKeyErrors(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := keyprovider.Ed25519KID(pub)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		provider fakeProvider
	}{
		{"provider failure", fakeProvider{err: errors.New("kms unavailable")}},
		{"wrong algorithm", fakeProvider{key: fakeSigningKey{kid: kid, algorithm: "RSA"}}},
		{"invalid kid", fakeProvider{key: fakeSigningKey{kid: "../bad", algorithm: "Ed25519"}}},
		{"sign failure", fakeProvider{key: fakeSigningKey{kid: kid, algorithm: "Ed25519", err: errors.New("sign failed")}}},
		{"short signature", fakeProvider{key: fakeSigningKey{kid: kid, algorithm: "Ed25519", signature: []byte("short")}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			signer, err := NewSigner(tc.provider)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, _, err := signer.Sign(context.Background(), "agent-1", 1, Settings{}, time.Now()); err == nil {
				t.Fatal("expected signing failure")
			}
		})
	}
}

func TestSignerCanonicalizesIssuedAtToUTCAndChecksum(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := keyprovider.Ed25519KID(pub)
	if err != nil {
		t.Fatal(err)
	}

	key := fakeSigningKey{kid: kid, algorithm: "Ed25519"}
	provider := dynamicProvider{fn: func(payload []byte) ([]byte, error) { return ed25519.Sign(priv, payload), nil }, kid: kid}
	signer, err := NewSigner(provider)
	if err != nil {
		t.Fatal(err)
	}

	loc := time.FixedZone("test", 3*60*60)
	issued := time.Date(2026, 8, 31, 20, 0, 0, 0, loc)
	env, payload, sig, err := signer.Sign(context.Background(), "agent-1", 9, Settings{FrequencyHz: 145800000, CaptureIntervalSeconds: 5}, issued)
	if err != nil {
		t.Fatal(err)
	}
	if env.KID != key.kid || len(sig) != ed25519.SignatureSize || len(payload) == 0 || env.Checksum == "" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

type dynamicProvider struct {
	fn  func([]byte) ([]byte, error)
	kid string
}

func (p dynamicProvider) ActiveSigningKey(context.Context) (keyprovider.SigningKey, error) {
	return dynamicKey{kid: p.kid, fn: p.fn}, nil
}
func (p dynamicProvider) SigningKey(context.Context, string) (keyprovider.SigningKey, error) {
	return dynamicKey{kid: p.kid, fn: p.fn}, nil
}

type dynamicKey struct {
	kid string
	fn  func([]byte) ([]byte, error)
}

func (k dynamicKey) KID() string                                      { return k.kid }
func (k dynamicKey) Algorithm() string                                { return "Ed25519" }
func (k dynamicKey) Sign(_ context.Context, p []byte) ([]byte, error) { return k.fn(p) }
