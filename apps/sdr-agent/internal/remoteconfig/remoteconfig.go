package remoteconfig

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/uplink"
)

type Settings struct {
	FrequencyHz            int64  `json:"frequency_hz"`
	CaptureIntervalSeconds int    `json:"capture_interval_seconds"`
	SatelliteID            string `json:"satellite_id,omitempty"`
}

type Payload struct {
	AgentID  string    `json:"agent_id"`
	Version  int64     `json:"version"`
	IssuedAt time.Time `json:"issued_at"`
	Config   Settings  `json:"config"`
}

type Verifier struct {
	keys    VerificationKeyProvider
	agentID string
}

func LoadVerifier(path, agentID string) (*Verifier, error) {
	return LoadVerifierWithKeyDirectory(path, "", agentID)
}

func LoadVerifierWithKeyDirectory(legacyPath, keyDir, agentID string) (*Verifier, error) {
	keys, err := loadVerificationKeyProvider(legacyPath, keyDir)
	if err != nil {
		return nil, err
	}
	return &Verifier{keys: keys, agentID: agentID}, nil
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
	var publicKey ed25519.PublicKey
	if strings.TrimSpace(envelope.KID) == "" {
		publicKey, err = v.keys.LegacyKey()
		if err != nil {
			return Payload{}, errors.New("remote config: legacy envelope has no configured legacy verification key")
		}
	} else {
		publicKey, err = v.keys.VerificationKey(envelope.KID)
		if err != nil {
			return Payload{}, errors.New("remote config: unknown signing key id")
		}
	}
	if !ed25519.Verify(publicKey, payloadBytes, signature) {
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
	if settings.SatelliteID != "" && !validUUID(settings.SatelliteID) {
		return errors.New("remote config: satellite id is invalid")
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

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
