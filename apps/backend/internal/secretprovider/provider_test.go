package secretprovider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileProviderRejectsLoosePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "JWT_SECRET")
	if err := os.WriteFile(path, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (FileProvider{Dir: dir}).Get(context.Background(), "JWT_SECRET"); err == nil {
		t.Fatal("expected permission rejection")
	}
}

func TestRemoteProviderRequiresHTTPS(t *testing.T) {
	if _, err := (RemoteProvider{Endpoint: "http://example.test"}).Get(context.Background(), "JWT_SECRET"); err == nil {
		t.Fatal("expected HTTPS rejection")
	}
}

func TestRemoteProviderRetrievesWithoutExposingToken(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer bootstrap" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"value": "managed-secret"})
	}))
	defer srv.Close()
	client := srv.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	p := RemoteProvider{Endpoint: srv.URL, Token: "bootstrap", Client: client}
	got, err := p.Get(context.Background(), "JWT_SECRET")
	if err != nil {
		t.Fatal(err)
	}
	if got != "managed-secret" {
		t.Fatalf("unexpected secret %q", got)
	}
	if !p.Health(context.Background()).Healthy {
		t.Fatal("expected healthy provider")
	}
	if p.Info().Location != srv.URL {
		t.Fatal("provider metadata mismatch")
	}
}
