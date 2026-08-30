package remoteconfig

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/uplink"
)

type Settings struct {
	FrequencyHz            int64 `json:"frequency_hz"`
	CaptureIntervalSeconds int   `json:"capture_interval_seconds"`
}

type Payload struct {
	AgentID  string    `json:"agent_id"`
	Version  int64     `json:"version"`
	IssuedAt time.Time `json:"issued_at"`
	Config   Settings  `json:"config"`
}

type Verifier struct {
	publicKey ed25519.PublicKey
	agentID   string
}

func LoadVerifier(path, agentID string) (*Verifier, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("remote config: read verification key: %w", err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("remote config: verification key is not public key PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("remote config: parse verification key: %w", err)
	}
	publicKey, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("remote config: verification key must be Ed25519")
	}
	return &Verifier{publicKey: publicKey, agentID: agentID}, nil
}

func (v *Verifier) Verify(envelope uplink.RemoteConfigEnvelope) (Payload, error) {
	payloadBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(envelope.Payload))
	if err != nil {
		return Payload{}, errors.New("remote config: invalid payload encoding")
	}
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(envelope.Signature))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return Payload{}, errors.New("remote config: invalid signature encoding")
	}
	checksum := sha256.Sum256(payloadBytes)
	if !strings.EqualFold(hex.EncodeToString(checksum[:]), strings.TrimSpace(envelope.Checksum)) {
		return Payload{}, errors.New("remote config: checksum mismatch")
	}
	if !ed25519.Verify(v.publicKey, payloadBytes, signature) {
		return Payload{}, errors.New("remote config: signature verification failed")
	}
	var payload Payload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return Payload{}, errors.New("remote config: invalid signed payload")
	}
	if payload.AgentID != v.agentID {
		return Payload{}, errors.New("remote config: agent identity mismatch")
	}
	if payload.Version <= 0 {
		return Payload{}, errors.New("remote config: invalid version")
	}
	if err := Validate(payload.Config); err != nil {
		return payload, err
	}
	return payload, nil
}

func Validate(settings Settings) error {
	if settings.FrequencyHz < 100000 || settings.FrequencyHz > 6000000000 {
		return errors.New("remote config: frequency out of bounds")
	}
	if settings.CaptureIntervalSeconds < 1 || settings.CaptureIntervalSeconds > 3600 {
		return errors.New("remote config: capture interval out of bounds")
	}
	return nil
}

type Manager struct {
	client   *uplink.Client
	verifier *Verifier
	version  int64
	apply    func(Settings) error
}

func NewManager(client *uplink.Client, verifier *Verifier, apply func(Settings) error) *Manager {
	return &Manager{client: client, verifier: verifier, apply: apply}
}

func (m *Manager) Version() int64 {
	return m.version
}

func (m *Manager) Poll() (bool, error) {
	envelope, changed, err := m.client.FetchRemoteConfig(m.version)
	if err != nil || !changed {
		return false, err
	}
	payload, err := m.verifier.Verify(envelope)
	if err != nil {
		if payload.Version > 0 {
			_ = m.client.AckRemoteConfig(uplink.RemoteConfigAck{Version: payload.Version, Status: "rejected", Message: err.Error(), Checksum: envelope.Checksum})
		}
		return false, err
	}
	if payload.Version <= m.version {
		return false, nil
	}
	if err := m.apply(payload.Config); err != nil {
		message := err.Error()
		if len(message) > 256 {
			message = message[:256]
		}
		if ackErr := m.client.AckRemoteConfig(uplink.RemoteConfigAck{Version: payload.Version, Status: "rejected", Message: message, Checksum: envelope.Checksum}); ackErr != nil {
			return false, fmt.Errorf("remote config: apply failed: %v; acknowledgement failed: %w", err, ackErr)
		}
		return false, err
	}
	if err := m.client.AckRemoteConfig(uplink.RemoteConfigAck{Version: payload.Version, Status: "applied", Checksum: envelope.Checksum}); err != nil {
		return false, err
	}
	m.version = payload.Version
	return true, nil
}
