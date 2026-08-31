package uplink

import (
	"crypto/ecdsa"
	"crypto/elliptic"
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
	"strings"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
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

func TestNewRejectsMissingCA(t *testing.T) {
	if _, err := New("https://example.invalid", "agent", "key", "", filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("expected missing CA error")
	}
}

func TestNewWithIdentityRequiresCertificateAndKey(t *testing.T) {
	if _, err := NewWithIdentity("https://example.invalid", "agent", "", "", "", "cert.pem", ""); err == nil {
		t.Fatal("expected incomplete identity error")
	}
	if _, err := NewWithIdentity("https://example.invalid", "agent", "", "", "", "", "key.pem"); err == nil {
		t.Fatal("expected incomplete identity error")
	}
}

func TestNewWithIdentityRejectsInvalidIdentity(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	if err := os.WriteFile(certPath, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewWithIdentity("https://example.invalid", "agent", "", "", "", certPath, keyPath); err == nil {
		t.Fatal("expected invalid identity error")
	}
}

func TestNewWithIdentityLoadsValidIdentity(t *testing.T) {
	certPath, keyPath := writeClientIdentity(t)
	client, err := NewWithIdentity("https://example.invalid", "agent", "legacy", "token", "", certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !client.mtls {
		t.Fatal("expected mTLS mode")
	}
	if client.transport == nil || client.transport.TLSClientConfig == nil || client.transport.TLSClientConfig.GetClientCertificate == nil {
		t.Fatal("client identity was not configured")
	}
	certificate, err := client.transport.TLSClientConfig.GetClientCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if certificate == nil || len(certificate.Certificate) == 0 {
		t.Fatal("expected loaded client certificate")
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

func TestReportDeviceNilIsNoop(t *testing.T) {
	client := &Client{}
	if err := client.ReportDevice(nil); err != nil {
		t.Fatal(err)
	}
}

func TestReportDeviceUsesBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agents/me/device" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("authorization = %q", got)
		}
		var report deviceReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			t.Fatal(err)
		}
		if report.Driver != "rtlsdr" || report.Serial != "ABC123" || !report.RX || report.TX {
			t.Fatalf("unexpected report: %+v", report)
		}
		if report.Capabilities.Architecture != "arm64" || report.Capabilities.FrequencyMaxHz != 1_766_000_000 || report.Capabilities.Channels != 1 {
			t.Fatalf("unexpected capabilities: %+v", report.Capabilities)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server, false)
	client.AuthToken = "user-token"
	info := &soapy.DeviceInfo{Driver: "rtlsdr", Hardware: "RTL2838", Manufacturer: "Vendor", Product: "SDR", Serial: "ABC123", RX: true, Capabilities: soapy.DeviceCapabilities{Architecture: "arm64", FrequencyMinHz: 24_000_000, FrequencyMaxHz: 1_766_000_000, SampleRatesHz: []int64{2_048_000}, Channels: 1}}
	if err := client.ReportDevice(info); err != nil {
		t.Fatal(err)
	}
}

func TestPostJSONRejectsUnserializablePayload(t *testing.T) {
	client := &Client{}
	if err := client.postJSON("/", make(chan int)); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestPostJSONRejectsInvalidURL(t *testing.T) {
	client := &Client{BackendURL: "://bad", http: http.DefaultClient}
	if err := client.postJSON("/test", struct{}{}); err == nil {
		t.Fatal("expected request creation error")
	}
}

func TestPostJSONReturnsBackendStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClient(server, false)
	if err := client.postJSON("/test", struct{}{}); err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostJSONReturnsTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := newTestClient(server, false)
	server.Close()
	if err := client.postJSON("/test", struct{}{}); err == nil {
		t.Fatal("expected transport error")
	}
}

func TestApplyAuthPrecedence(t *testing.T) {
	tests := []struct {
		name           string
		mtls           bool
		apiKey         string
		token          string
		expectAgentKey string
		expectBearer   string
	}{
		{name: "mtls suppresses credentials", mtls: true, apiKey: "key", token: "token"},
		{name: "api key before bearer", apiKey: "key", token: "token", expectAgentKey: "key"},
		{name: "bearer fallback", token: "token", expectBearer: "Bearer token"},
		{name: "no credentials"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{mtls: tc.mtls, APIKey: tc.apiKey, AuthToken: tc.token}
			req := httptest.NewRequest(http.MethodGet, "http://example.test", nil)
			client.applyAuth(req)
			if got := req.Header.Get("X-Agent-Key"); got != tc.expectAgentKey {
				t.Fatalf("agent key = %q", got)
			}
			if got := req.Header.Get("Authorization"); got != tc.expectBearer {
				t.Fatalf("authorization = %q", got)
			}
		})
	}
}

func TestReloadIdentity(t *testing.T) {
	client := &Client{}
	client.ReloadIdentity()
	client.transport = &http.Transport{}
	client.ReloadIdentity()
}

func TestFetchRemoteConfigRequiresMTLS(t *testing.T) {
	client := &Client{}
	if _, _, err := client.FetchRemoteConfig(0); err == nil {
		t.Fatal("expected mTLS requirement")
	}
}

func TestFetchRemoteConfigNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("after_version"); got != "42" {
			t.Fatalf("after_version = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server, true)
	_, found, err := client.FetchRemoteConfig(42)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("expected no configuration")
	}
}

func TestFetchRemoteConfigReturnsEnvelope(t *testing.T) {
	expected := RemoteConfigEnvelope{Payload: `{"version":2}`, Signature: "sig", Checksum: "sum"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(expected); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	client := newTestClient(server, true)
	envelope, found, err := client.FetchRemoteConfig(1)
	if err != nil {
		t.Fatal(err)
	}
	if !found || envelope != expected {
		t.Fatalf("unexpected envelope: found=%v envelope=%+v", found, envelope)
	}
}

func TestFetchRemoteConfigRejectsBackendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(server, true)
	if _, _, err := client.FetchRemoteConfig(0); err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchRemoteConfigRejectsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer server.Close()

	client := newTestClient(server, true)
	if _, _, err := client.FetchRemoteConfig(0); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestFetchRemoteConfigRejectsInvalidURL(t *testing.T) {
	client := &Client{BackendURL: "://bad", mtls: true, http: http.DefaultClient}
	if _, _, err := client.FetchRemoteConfig(0); err == nil {
		t.Fatal("expected URL error")
	}
}

func TestFetchRemoteConfigReturnsTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := newTestClient(server, true)
	server.Close()
	if _, _, err := client.FetchRemoteConfig(0); err == nil {
		t.Fatal("expected transport error")
	}
}

func TestAckRemoteConfigRequiresMTLS(t *testing.T) {
	client := &Client{}
	if err := client.AckRemoteConfig(RemoteConfigAck{}); err == nil {
		t.Fatal("expected mTLS requirement")
	}
}

func TestAckRemoteConfigPostsAcknowledgementWithoutLegacyAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agents/me/config/ack" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("X-Agent-Key") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("legacy credentials must not be sent in mTLS mode")
		}
		var ack RemoteConfigAck
		if err := json.NewDecoder(r.Body).Decode(&ack); err != nil {
			t.Fatal(err)
		}
		if ack.Version != 7 || ack.Status != "applied" || ack.Checksum != "abc" {
			t.Fatalf("unexpected acknowledgement: %+v", ack)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server, true)
	client.APIKey = "legacy-key"
	client.AuthToken = "legacy-token"
	if err := client.AckRemoteConfig(RemoteConfigAck{Version: 7, Status: "applied", Checksum: "abc"}); err != nil {
		t.Fatal(err)
	}
}

func newTestClient(server *httptest.Server, mtls bool) *Client {
	return &Client{BackendURL: server.URL, http: server.Client(), mtls: mtls}
}

func writeClientIdentity(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "agent-test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

func TestSendIncludesConfiguredSatelliteID(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["satellite_id"] != "550e8400-e29b-41d4-a716-446655440000" {
			t.Fatalf("satellite_id = %v", payload["satellite_id"])
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	client := newTestClient(server, true)
	client.SetSatelliteID("550e8400-e29b-41d4-a716-446655440000")
	if err := client.Send(capture.Reading{FrequencyHz: 437500000, PowerDbm: -80, CapturedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
}
