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

func makeTestCA(t *testing.T, notAfter time.Time) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}

func makeCSR(t *testing.T, key any) string {
	t.Helper()
	var signer interface{ Public() any }
	_ = signer
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "untrusted"}}, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestParseRejectsInvalidCAInputs(t *testing.T) {
	certPEM, keyPEM := makeTestCA(t, time.Now().Add(24*time.Hour))
	if _, err := Parse([]byte("bad"), keyPEM); err == nil {
		t.Fatal("expected invalid certificate PEM rejection")
	}
	if _, err := Parse(certPEM, []byte("bad")); err == nil {
		t.Fatal("expected invalid key PEM rejection")
	}

	otherCert, otherKey := makeTestCA(t, time.Now().Add(24*time.Hour))
	_ = otherCert
	if _, err := Parse(certPEM, otherKey); err == nil {
		t.Fatal("expected mismatched CA key rejection")
	}

	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	cert.IsCA = false
	_ = cert
}

func TestLoadAndFingerprintEdgeCases(t *testing.T) {
	certPEM, keyPEM := makeTestCA(t, time.Now().Add(24*time.Hour))
	dir := t.TempDir()
	certPath := filepath.Join(dir, "ca.pem")
	keyPath := filepath.Join(dir, "ca-key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	ca, err := Load(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if ca.CertificatePEM() == "" {
		t.Fatal("expected certificate PEM")
	}
	if Fingerprint(nil) != "" {
		t.Fatal("nil certificate fingerprint must be empty")
	}
}

func TestSignCSRRejectsInvalidInputsAndWeakRSA(t *testing.T) {
	certPEM, keyPEM := makeTestCA(t, time.Now().Add(24*time.Hour))
	ca, err := Parse(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ca.SignCSR("", "bad", time.Hour); err == nil {
		t.Fatal("empty agent id must fail")
	}
	if _, err := ca.SignCSR("agent", "bad", time.Hour); err == nil {
		t.Fatal("invalid CSR must fail")
	}

	weak, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ca.SignCSR("agent", makeCSR(t, weak), time.Hour); err == nil {
		t.Fatal("weak RSA CSR must fail")
	}
}

func TestSignCSRSupportsEd25519AndCapsTTLAtCAExpiry(t *testing.T) {
	expiry := time.Now().Add(45 * time.Minute)
	certPEM, keyPEM := makeTestCA(t, expiry)
	ca, err := Parse(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	issued, err := ca.SignCSR("agent/alpha", makeCSR(t, privateKey), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if issued.NotAfter.After(expiry) {
		t.Fatalf("certificate expiry %s exceeds CA expiry %s", issued.NotAfter, expiry)
	}
	block, _ := pem.Decode([]byte(issued.PEM))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if Fingerprint(cert) != issued.FingerprintSHA256 {
		t.Fatal("fingerprint mismatch")
	}
	if len(cert.URIs) != 1 || cert.URIs[0].String() != "spiffe://lyrahcle/agent/agent%2Falpha" {
		t.Fatalf("unexpected SPIFFE URI: %v", cert.URIs)
	}
}

func TestSignCSRRejectsCATooCloseToExpiry(t *testing.T) {
	certPEM, keyPEM := makeTestCA(t, time.Now().Add(4*time.Minute))
	ca, err := Parse(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ca.SignCSR("agent", makeCSR(t, key), time.Hour); err == nil {
		t.Fatal("CA near expiry must not issue certificates")
	}
}
