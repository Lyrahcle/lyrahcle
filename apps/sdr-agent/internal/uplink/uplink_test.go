package uplink

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
)

func TestNewRejectsInvalidCA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New("https://example.invalid", "agent", "key", "", path); err == nil {
		t.Fatal("expected invalid CA error")
	}
}

func TestClientUsesCustomCAAndAgentKey(t *testing.T) {
	var received bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/rf/ingest" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("X-Agent-Key") != "agent-secret" {
			t.Fatalf("missing agent key")
		}
		received = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	cert, err := x509.ParseCertificate(server.Certificate().Raw)
	if err != nil {
		t.Fatal(err)
	}
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if err := os.WriteFile(caPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	client, err := New(server.URL, "agent-01", "agent-secret", "", caPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Send(capture.Reading{FrequencyHz: 437500000, PowerDbm: -80, CapturedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if !received {
		t.Fatal("server did not receive request")
	}
}
