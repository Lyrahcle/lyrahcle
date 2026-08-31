package agentidentity

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func caCertificateForSigner(t *testing.T, signer any, isCA bool, usage x509.KeyUsage) []byte {
	t.Helper()
	var public any
	switch key := signer.(type) {
	case *ecdsa.PrivateKey:
		public = &key.PublicKey
	case *rsa.PrivateKey:
		public = &key.PublicKey
	case ed25519.PrivateKey:
		public = key.Public()
	default:
		t.Fatalf("unsupported test signer %T", signer)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(99),
		Subject:               pkix.Name{CommonName: "coverage-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              usage,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, public, signer)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestParseSignerSupportsLegacyECAndPKCS1Formats(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}
	ecSigner, err := parseSigner(ecDER)
	if err != nil {
		t.Fatal(err)
	}
	if !publicKeysEqual(ecSigner.Public(), &ecKey.PublicKey) {
		t.Fatal("EC signer public key mismatch")
	}

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaSigner, err := parseSigner(x509.MarshalPKCS1PrivateKey(rsaKey))
	if err != nil {
		t.Fatal(err)
	}
	if !publicKeysEqual(rsaSigner.Public(), &rsaKey.PublicKey) {
		t.Fatal("RSA signer public key mismatch")
	}

	if _, err := parseSigner([]byte("not-a-private-key")); err == nil {
		t.Fatal("unsupported DER must fail")
	}
}

func TestParseRejectsCertificateWithoutCASigningUsage(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	for _, tc := range []struct {
		name  string
		isCA  bool
		usage x509.KeyUsage
	}{
		{"not ca", false, x509.KeyUsageDigitalSignature},
		{"ca without cert sign", true, x509.KeyUsageDigitalSignature},
	} {
		t.Run(tc.name, func(t *testing.T) {
			certPEM := caCertificateForSigner(t, key, tc.isCA, tc.usage)
			if _, err := Parse(certPEM, keyPEM); err == nil {
				t.Fatal("non-signing CA certificate must fail")
			}
		})
	}
}

func TestLoadReportsMissingCertificateAndKey(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(filepath.Join(dir, "missing-cert.pem"), filepath.Join(dir, "missing-key.pem")); err == nil {
		t.Fatal("missing certificate must fail")
	}

	certPEM, _ := makeTestCA(t, time.Now().Add(24*time.Hour))
	certPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(certPath, filepath.Join(dir, "missing-key.pem")); err == nil {
		t.Fatal("missing key must fail")
	}
}

func TestAcceptablePublicKeyPolicy(t *testing.T) {
	ec256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if !acceptablePublicKey(&ec256.PublicKey) {
		t.Fatal("P-256 must be accepted")
	}

	ec224, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if acceptablePublicKey(&ec224.PublicKey) {
		t.Fatal("P-224 must be rejected")
	}

	rsa2048, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if !acceptablePublicKey(&rsa2048.PublicKey) {
		t.Fatal("RSA-2048 must be accepted")
	}

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if !acceptablePublicKey(pub) {
		t.Fatal("Ed25519 must be accepted")
	}
	if acceptablePublicKey(ed25519.PublicKey([]byte("short"))) {
		t.Fatal("malformed Ed25519 public key must be rejected")
	}
	if acceptablePublicKey("unsupported") {
		t.Fatal("unsupported key types must be rejected")
	}
}

func TestPublicKeysEqualRejectsUnmarshalableValues(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if publicKeysEqual(make(chan int), &key.PublicKey) {
		t.Fatal("invalid left public key must compare false")
	}
	if publicKeysEqual(&key.PublicKey, make(chan int)) {
		t.Fatal("invalid right public key must compare false")
	}
}

func TestSignCSRWithRSA2048CarriesClientIdentityFields(t *testing.T) {
	certPEM, keyPEM := makeTestCA(t, time.Now().Add(24*time.Hour))
	ca, err := Parse(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := ca.SignCSR("  agent-rsa  ", makeCSR(t, key), 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode([]byte(issued.PEM))
	if block == nil {
		t.Fatal("issued certificate PEM missing")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if cert.Subject.CommonName != "agent-rsa" {
		t.Fatalf("common name=%q want agent-rsa", cert.Subject.CommonName)
	}
	if len(cert.Subject.Organization) != 1 || cert.Subject.Organization[0] != "Lyrahcle SDR Agent" {
		t.Fatalf("organization=%v", cert.Subject.Organization)
	}
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Fatal("digital-signature key usage missing")
	}
	if len(cert.ExtKeyUsage) != 1 || cert.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth {
		t.Fatalf("extended key usage=%v", cert.ExtKeyUsage)
	}
	if issued.SerialNumber == "" || issued.FingerprintSHA256 == "" {
		t.Fatal("issued certificate metadata incomplete")
	}
}

func TestSignCSRRejectsNonPositiveTTL(t *testing.T) {
	certPEM, keyPEM := makeTestCA(t, time.Now().Add(24*time.Hour))
	ca, err := Parse(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, ttl := range []time.Duration{0, -time.Minute} {
		if _, err := ca.SignCSR("agent", makeCSR(t, key), ttl); err == nil {
			t.Fatalf("ttl %s must fail", ttl)
		}
	}
}
