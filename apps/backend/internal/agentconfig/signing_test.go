package agentconfig

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
)

func TestSignerProducesVerifiableEnvelope(t *testing.T) {
	privatePEM, publicPEM, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "signing.pem")
	if err := os.WriteFile(path, privatePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	signer, err := LoadSigner(path)
	if err != nil {
		t.Fatal(err)
	}
	envelope, payload, signature, err := signer.Sign(context.Background(), "sdr-1", 7, Settings{FrequencyHz: 437500000, CaptureIntervalSeconds: 10}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(publicPEM)
	if block == nil {
		t.Fatal("missing public key PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := parsed.(ed25519.PublicKey)
	expectedKID, err := keyprovider.Ed25519KID(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.KID != expectedKID {
		t.Fatalf("envelope kid = %q, want %q", envelope.KID, expectedKID)
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		t.Fatal("signature did not verify")
	}
	decodedPayload, err := base64.RawURLEncoding.DecodeString(envelope.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if string(decodedPayload) != string(payload) {
		t.Fatal("envelope payload mismatch")
	}
}
