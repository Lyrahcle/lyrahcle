package identity

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureRequiresHTTPSForEnrollment(t *testing.T) {
	dir := t.TempDir()
	err := Ensure(Config{
		BackendURL:      "http://example.test",
		AgentID:         "agent-1",
		EnrollmentToken: "abcdefghijklmnopqrstuvwxyz0123456789",
		CertificatePath: filepath.Join(dir, "agent.crt"),
		PrivateKeyPath:  filepath.Join(dir, "agent.key"),
	})
	if err == nil {
		t.Fatal("expected HTTPS requirement error")
	}
}

func TestEnsureEnrollsAndPersistsIdentity(t *testing.T) {
	var sawCSR bool
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agents/enroll" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["token"] != "abcdefghijklmnopqrstuvwxyz0123456789" {
			t.Fatalf("unexpected token")
		}
		if body["csr_pem"] == "" {
			t.Fatalf("missing CSR")
		}
		sawCSR = true
		csrBlock, _ := pem.Decode([]byte(body["csr_pem"]))
		csr, err := x509.ParseCertificateRequest(csrBlock.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject:      pkix.Name{CommonName: "agent-1"},
			NotBefore:    time.Now().Add(-time.Minute),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, server.Certificate(), csr.PublicKey, server.TLS.Certificates[0].PrivateKey)
		if err != nil {
			t.Fatal(err)
		}
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		json.NewEncoder(w).Encode(map[string]any{
			"agent_id":        "agent-1",
			"certificate_pem": string(certPEM),
			"expires_at":      time.Now().Add(time.Hour),
		})
	}))
	defer server.Close()

	cert, err := x509.ParseCertificate(server.Certificate().Raw)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	caPath := filepath.Join(dir, "server-ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(dir, "identity", "agent.crt")
	keyPath := filepath.Join(dir, "identity", "agent.key")
	if err := Ensure(Config{
		BackendURL:      server.URL,
		AgentID:         "agent-1",
		EnrollmentToken: "abcdefghijklmnopqrstuvwxyz0123456789",
		ServerCAPath:    caPath,
		CertificatePath: certPath,
		PrivateKeyPath:  keyPath,
		AgentVersion:    "v9.0",
		FirmwareVersion: "fw-1",
		RotateBefore:    time.Hour,
	}); err != nil {
		t.Fatal(err)
	}
	if !sawCSR {
		t.Fatal("server did not receive CSR")
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private key mode = %o", info.Mode().Perm())
	}
	if _, err := os.Stat(certPath); err != nil {
		t.Fatal(err)
	}
}
