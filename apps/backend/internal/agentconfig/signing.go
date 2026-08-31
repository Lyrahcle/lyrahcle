package agentconfig

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
)

type Settings struct {
	FrequencyHz            int64      `json:"frequency_hz"`
	CaptureIntervalSeconds int        `json:"capture_interval_seconds"`
	SatelliteID            *uuid.UUID `json:"satellite_id,omitempty"`
}

type Payload struct {
	AgentID  string    `json:"agent_id"`
	Version  int64     `json:"version"`
	IssuedAt time.Time `json:"issued_at"`
	Config   Settings  `json:"config"`
}

type Envelope struct {
	KID       string `json:"kid,omitempty"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	Checksum  string `json:"checksum_sha256"`
}

type Signer struct {
	provider keyprovider.Provider
}

func NewSigner(provider keyprovider.Provider) (*Signer, error) {
	if provider == nil {
		return nil, errors.New("agent config signing key provider is required")
	}
	return &Signer{provider: provider}, nil
}

// LoadSigner preserves the existing file-based configuration while routing all
// private-key use through the KeyProvider abstraction.
func LoadSigner(path string) (*Signer, error) {
	provider, err := keyprovider.NewLocalProvider(path)
	if err != nil {
		return nil, fmt.Errorf("agent config signing key: %w", err)
	}
	return NewSigner(provider)
}

func GenerateKeyPair() ([]byte, []byte, error) {
	return keyprovider.GenerateEd25519KeyPair()
}

func (s *Signer) Sign(ctx context.Context, agentID string, version int64, settings Settings, issuedAt time.Time) (Envelope, []byte, []byte, error) {
	payloadBytes, err := json.Marshal(Payload{
		AgentID:  agentID,
		Version:  version,
		IssuedAt: issuedAt.UTC(),
		Config:   settings,
	})
	if err != nil {
		return Envelope{}, nil, nil, err
	}
	key, err := s.provider.ActiveSigningKey(ctx)
	if err != nil {
		return Envelope{}, nil, nil, fmt.Errorf("agent config signing key: %w", err)
	}
	if key.Algorithm() != "Ed25519" {
		return Envelope{}, nil, nil, fmt.Errorf("agent config signing key: unsupported algorithm %q", key.Algorithm())
	}
	if err := keyprovider.ValidateKID(key.KID()); err != nil {
		return Envelope{}, nil, nil, fmt.Errorf("agent config signing key: %w", err)
	}
	signature, err := key.Sign(ctx, payloadBytes)
	if err != nil {
		return Envelope{}, nil, nil, fmt.Errorf("agent config signing key: %w", err)
	}
	if len(signature) != ed25519.SignatureSize {
		return Envelope{}, nil, nil, errors.New("agent config signing key returned invalid Ed25519 signature")
	}
	checksum := sha256.Sum256(payloadBytes)
	envelope := Envelope{
		KID:       key.KID(),
		Payload:   base64.RawURLEncoding.EncodeToString(payloadBytes),
		Signature: base64.RawURLEncoding.EncodeToString(signature),
		Checksum:  hex.EncodeToString(checksum[:]),
	}
	return envelope, payloadBytes, signature, nil
}
