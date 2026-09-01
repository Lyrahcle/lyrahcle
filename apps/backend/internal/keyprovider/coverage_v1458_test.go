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
	"testing"
	"time"
)

func TestNilProvidersFailClosed(t *testing.T) {
	ctx := context.Background()

	var local *LocalProvider
	if _, err := local.ActiveSigningKey(ctx); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("nil local ActiveSigningKey error=%v", err)
	}
	if _, err := local.SigningKey(ctx, "kid"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("nil local SigningKey error=%v", err)
	}

	var managed *ManagedLocalProvider
	if _, err := managed.RotationPolicy(ctx); !errors.Is(err, ErrLifecycleUnsupported) {
		t.Fatalf("nil managed RotationPolicy error=%v", err)
	}
	if _, err := managed.UpdateRotationPolicy(ctx, DefaultRotationPolicy()); !errors.Is(err, ErrLifecycleUnsupported) {
		t.Fatalf("nil managed UpdateRotationPolicy error=%v", err)
	}
}

func TestLocalKeyRejectsInvalidPrivateKeyLength(t *testing.T) {
	key := &localEd25519Key{kid: "test", privateKey: ed25519.PrivateKey([]byte("short"))}
	if _, err := key.Sign(context.Background(), []byte("payload")); err == nil {
		t.Fatal("invalid private key length must fail signing")
	}
}

func TestGenerateEd25519KeyPairRoundTrips(t *testing.T) {
	privatePEM, publicPEM, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatal(err)
	}
	privateBlock, rest := pem.Decode(privatePEM)
	if privateBlock == nil || privateBlock.Type != "PRIVATE KEY" || len(rest) != 0 {
		t.Fatal("invalid generated private PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		t.Fatalf("generated key type=%T want Ed25519", parsed)
	}
	publicBlock, rest := pem.Decode(publicPEM)
	if publicBlock == nil || publicBlock.Type != "PUBLIC KEY" || len(rest) != 0 {
		t.Fatal("invalid generated public PEM")
	}
	parsedPublic, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, ok := parsedPublic.(ed25519.PublicKey)
	if !ok || !publicKey.Equal(privateKey.Public()) {
		t.Fatal("generated public/private key pair does not match")
	}
}

func TestManagedProviderImportsBootstrapKey(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := NewManagedLocalProvider(t.TempDir(), writePrivateKey(t, privateKey))
	if err != nil {
		t.Fatal(err)
	}
	active, err := provider.ActiveSigningKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	kid, err := Ed25519KID(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if active.KID() != kid {
		t.Fatalf("active KID=%q want %q", active.KID(), kid)
	}
}

func TestManagedProviderLifecycleErrorBranches(t *testing.T) {
	ctx := context.Background()
	provider, err := NewManagedLocalProvider(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	active, err := provider.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := provider.Activate(ctx, "missing", time.Hour); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("missing activate error=%v", err)
	}
	if _, err := provider.Activate(ctx, active.KID(), time.Hour); !errors.Is(err, ErrKeyNotStaged) {
		t.Fatalf("active activate error=%v", err)
	}
	if _, err := provider.Activate(ctx, active.KID(), 59*time.Minute); err == nil {
		t.Fatal("short grace period must fail")
	}
	if _, err := provider.Activate(ctx, active.KID(), 31*24*time.Hour); err == nil {
		t.Fatal("excessive grace period must fail")
	}
	if _, err := provider.Revoke(ctx, "missing"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("missing revoke error=%v", err)
	}
	if _, err := provider.PublicKeyPEM(ctx, "missing"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("missing public key error=%v", err)
	}

	staged, _, err := provider.PrepareRotation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := provider.PrepareRotation(ctx); !errors.Is(err, ErrRotationPending) {
		t.Fatalf("second staged rotation error=%v want ErrRotationPending", err)
	}

	revoked, err := provider.Revoke(ctx, staged.KID)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Status != KeyStatusRevoked {
		t.Fatalf("staged revoke status=%s", revoked.Status)
	}
	again, err := provider.Revoke(ctx, staged.KID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != KeyStatusRevoked || again.RevokedAt == nil {
		t.Fatalf("idempotent revoke metadata=%+v", again)
	}
}

func TestManagedProviderPublicKeyPEMMatchesActiveSigner(t *testing.T) {
	ctx := context.Background()
	provider, err := NewManagedLocalProvider(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	active, err := provider.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := provider.PublicKeyPEM(ctx, active.KID())
	if err != nil {
		t.Fatal(err)
	}
	block, rest := pem.Decode(raw)
	if block == nil || block.Type != "PUBLIC KEY" || len(rest) != 0 {
		t.Fatal("invalid public key PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, ok := parsed.(ed25519.PublicKey)
	if !ok {
		t.Fatalf("public key type=%T want Ed25519", parsed)
	}
	payload := []byte("verification-payload")
	signature, err := active.Sign(ctx, payload)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		t.Fatal("exported public key did not verify active signer")
	}
}

func TestManagedProviderRejectsInvalidManifestShapes(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewManagedLocalProvider(dir, ""); err == nil {
			t.Fatal("invalid JSON manifest must fail")
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		dir := t.TempDir()
		raw, err := json.Marshal(managedManifest{Version: 2, Keys: map[string]managedKeyRecord{"x": {FileName: "x.pem"}}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewManagedLocalProvider(dir, ""); err == nil {
			t.Fatal("unsupported manifest version must fail")
		}
	})

	t.Run("invalid kid", func(t *testing.T) {
		dir := t.TempDir()
		raw, err := json.Marshal(managedManifest{
			Version: 1,
			Keys: map[string]managedKeyRecord{
				"../bad": {FileName: "key.pem", Status: KeyStatusActive},
			},
			RotationPolicy: DefaultRotationPolicy(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewManagedLocalProvider(dir, ""); err == nil {
			t.Fatal("invalid manifest KID must fail")
		}
	})
}

func TestLocalProviderRejectsEmptyActivePathAndNonEd25519Key(t *testing.T) {
	if _, err := NewLocalProvider(" "); err == nil {
		t.Fatal("empty active path must fail")
	}

	_, rsaLikePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	path := writePrivateKey(t, rsaLikePrivate)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, data...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLocalProvider(path); err == nil {
		t.Fatal("multiple PEM blocks must fail")
	}
}
