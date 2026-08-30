package remoteconfig

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/uplink"
)

func TestVerifierAcceptsSignedConfig(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "verify.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0o644); err != nil {
		t.Fatal(err)
	}
	verifier, err := LoadVerifier(path, "sdr-1")
	if err != nil {
		t.Fatal(err)
	}
	payloadBytes, err := json.Marshal(Payload{AgentID: "sdr-1", Version: 3, IssuedAt: time.Unix(100, 0).UTC(), Config: Settings{FrequencyHz: 437500000, CaptureIntervalSeconds: 12}})
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, payloadBytes)
	checksum := sha256.Sum256(payloadBytes)
	envelope := uplink.RemoteConfigEnvelope{
		Payload:   base64.RawURLEncoding.EncodeToString(payloadBytes),
		Signature: base64.RawURLEncoding.EncodeToString(signature),
		Checksum:  hex.EncodeToString(checksum[:]),
	}
	payload, err := verifier.Verify(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Version != 3 || payload.Config.FrequencyHz != 437500000 || payload.Config.CaptureIntervalSeconds != 12 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestVerifierRejectsWrongAgentAndTampering(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "verify.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0o644); err != nil {
		t.Fatal(err)
	}
	verifier, err := LoadVerifier(path, "sdr-1")
	if err != nil {
		t.Fatal(err)
	}
	payloadBytes, err := json.Marshal(Payload{AgentID: "sdr-2", Version: 1, IssuedAt: time.Now().UTC(), Config: Settings{FrequencyHz: 437500000, CaptureIntervalSeconds: 10}})
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, payloadBytes)
	checksum := sha256.Sum256(payloadBytes)
	envelope := uplink.RemoteConfigEnvelope{Payload: base64.RawURLEncoding.EncodeToString(payloadBytes), Signature: base64.RawURLEncoding.EncodeToString(signature), Checksum: hex.EncodeToString(checksum[:])}
	if _, err := verifier.Verify(envelope); err == nil {
		t.Fatal("expected identity mismatch")
	}
	envelope.Payload = base64.RawURLEncoding.EncodeToString(append(payloadBytes, 'x'))
	if _, err := verifier.Verify(envelope); err == nil {
		t.Fatal("expected tampering rejection")
	}
}

func TestValidateSatelliteID(t *testing.T) {
	valid := Settings{FrequencyHz: 437500000, CaptureIntervalSeconds: 10, SatelliteID: "550e8400-e29b-41d4-a716-446655440000"}
	if err := Validate(valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.SatelliteID = "not-a-uuid"
	if err := Validate(invalid); err == nil {
		t.Fatal("expected invalid satellite id to be rejected")
	}
}
