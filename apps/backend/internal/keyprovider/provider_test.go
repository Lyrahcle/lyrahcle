package keyprovider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writePrivateKey(t *testing.T, privateKey ed25519.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "signing.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLocalProviderSignsWithoutExposingPrivateKey(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewLocalProvider(writePrivateKey(t, privateKey))
	if err != nil {
		t.Fatal(err)
	}
	key, err := provider.ActiveSigningKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expectedKID, err := Ed25519KID(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if key.KID() != expectedKID {
		t.Fatalf("kid = %q, want %q", key.KID(), expectedKID)
	}
	payload := []byte("signed-config")
	signature, err := key.Sign(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		t.Fatal("signature did not verify")
	}
	resolved, err := provider.SigningKey(context.Background(), expectedKID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.KID() != expectedKID {
		t.Fatalf("resolved kid = %q", resolved.KID())
	}
}

func TestLocalProviderRejectsUnknownKID(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewLocalProvider(writePrivateKey(t, privateKey))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.SigningKey(context.Background(), "ed25519-sha256-missing"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("error = %v, want ErrKeyNotFound", err)
	}
}

func TestLocalProviderResolvesActiveAndPreviousKeysByKID(t *testing.T) {
	activePublic, activePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	previousPublic, previousPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewLocalProvider(writePrivateKey(t, activePrivate), writePrivateKey(t, previousPrivate))
	if err != nil {
		t.Fatal(err)
	}
	active, err := provider.ActiveSigningKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	activeKID, _ := Ed25519KID(activePublic)
	previousKID, _ := Ed25519KID(previousPublic)
	if active.KID() != activeKID {
		t.Fatalf("active kid = %q, want %q", active.KID(), activeKID)
	}
	previous, err := provider.SigningKey(context.Background(), previousKID)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("historical-config")
	signature, err := previous.Sign(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(previousPublic, payload, signature) {
		t.Fatal("previous signing key did not verify")
	}
}

func TestManagedProviderStagesActivatesAndRevokes(t *testing.T) {
	ctx := context.Background()
	provider, err := NewManagedLocalProvider(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	initial, err := provider.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged, publicPEM, err := provider.PrepareRotation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if staged.Status != KeyStatusStaged || staged.KID == initial.KID() || len(publicPEM) == 0 {
		t.Fatalf("staged=%#v public=%d", staged, len(publicPEM))
	}
	if _, err := provider.SigningKey(ctx, staged.KID); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("staged key resolved before activation: %v", err)
	}
	activated, err := provider.Activate(ctx, staged.KID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if activated.Status != KeyStatusActive {
		t.Fatalf("activated status=%s", activated.Status)
	}
	active, err := provider.ActiveSigningKey(ctx)
	if err != nil || active.KID() != staged.KID {
		t.Fatalf("active=%v err=%v", active, err)
	}
	if _, err := provider.SigningKey(ctx, initial.KID()); err != nil {
		t.Fatalf("previous active key should remain trusted during grace: %v", err)
	}
	if _, err := provider.Revoke(ctx, initial.KID()); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.SigningKey(ctx, initial.KID()); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("revoked key error=%v want ErrKeyRevoked", err)
	}
	if _, err := provider.Revoke(ctx, staged.KID); !errors.Is(err, ErrActiveKey) {
		t.Fatalf("active revoke error=%v want ErrActiveKey", err)
	}
}

func TestManagedProviderPersistsLifecycleAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	provider, err := NewManagedLocalProvider(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	staged, _, err := provider.PrepareRotation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Activate(ctx, staged.KID, 2*time.Hour); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewManagedLocalProvider(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	active, err := reloaded.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if active.KID() != staged.KID {
		t.Fatalf("active kid after restart=%q want %q", active.KID(), staged.KID)
	}
	keys, err := reloaded.Keys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0].Status != KeyStatusActive {
		t.Fatalf("keys after restart=%#v", keys)
	}
}

func TestManagedProviderRejectsSymlinkDirectory(t *testing.T) {
	realDir := t.TempDir()
	link := filepath.Join(t.TempDir(), "keys")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := NewManagedLocalProvider(link, ""); err == nil {
		t.Fatal("expected symlink key directory to be rejected")
	}
}

func TestLocalProviderRejectsMissingAndMalformedKeyFiles(t *testing.T) {
	if _, err := NewLocalProvider(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("expected missing key file to be rejected")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(path, []byte("not a pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLocalProvider(path); err == nil {
		t.Fatal("expected malformed PEM to be rejected")
	}
}

func TestLocalProviderRejectsSymlinkKeyFile(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	realPath := writePrivateKey(t, privateKey)
	link := filepath.Join(t.TempDir(), "signing-key.pem")
	if err := os.Symlink(realPath, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := NewLocalProvider(link); err == nil {
		t.Fatal("expected symlink signing key to be rejected")
	}
}

func TestManagedProviderRejectsTamperedManifestFilename(t *testing.T) {
	dir := t.TempDir()
	if _, err := NewManagedLocalProvider(dir, ""); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest managedManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for kid, rec := range manifest.Keys {
		rec.FileName = "../escape.pem"
		manifest.Keys[kid] = rec
		break
	}
	raw, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewManagedLocalProvider(dir, ""); err == nil {
		t.Fatal("expected path traversal filename in manifest to be rejected")
	}
}

func TestManagedProviderAutomaticallyRevokesExpiredGraceKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	provider, err := NewManagedLocalProvider(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	initial, err := provider.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged, _, err := provider.PrepareRotation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Activate(ctx, staged.KID, time.Hour); err != nil {
		t.Fatal(err)
	}

	provider.mu.Lock()
	rec := provider.manifest.Keys[initial.KID()]
	expired := time.Now().UTC().Add(-time.Minute)
	rec.Status = KeyStatusGrace
	rec.GraceExpiresAt = &expired
	provider.manifest.Keys[initial.KID()] = rec
	if err := provider.saveManifestLocked(); err != nil {
		provider.mu.Unlock()
		t.Fatal(err)
	}
	provider.mu.Unlock()

	if _, err := provider.SigningKey(ctx, initial.KID()); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("expired grace key error=%v want ErrKeyRevoked", err)
	}
	reloaded, err := NewManagedLocalProvider(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := reloaded.Keys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if key.KID == initial.KID() && (key.Status != KeyStatusRevoked || key.RevokedAt == nil) {
			t.Fatalf("expired key metadata=%+v want revoked with timestamp", key)
		}
	}
}

func TestValidateKIDAndEd25519KIDRejectInvalidInput(t *testing.T) {
	for _, kid := range []string{"", "contains space", "../escape", strings.Repeat("a", 129)} {
		if err := ValidateKID(kid); err == nil {
			t.Fatalf("ValidateKID(%q) unexpectedly succeeded", kid)
		}
	}
	if _, err := Ed25519KID(make(ed25519.PublicKey, ed25519.PublicKeySize-1)); err == nil {
		t.Fatal("expected invalid public key length to be rejected")
	}
}
