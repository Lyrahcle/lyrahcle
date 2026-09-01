package secretprovider

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnvProviderContracts(t *testing.T) {
	const key = "LYRAHCLE_TEST_SECRET"
	t.Setenv(key, "managed-value")
	provider := EnvProvider{}
	got, err := provider.Get(context.Background(), key)
	if err != nil || got != "managed-value" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if info := provider.Info(); info.Name != "env" || info.External || info.Location != "process environment" {
		t.Fatalf("Info = %#v", info)
	}
	if health := provider.Health(context.Background()); !health.Healthy {
		t.Fatalf("Health = %#v", health)
	}

	t.Setenv(key, "")
	if _, err := provider.Get(context.Background(), key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty Get error = %v, want ErrNotFound", err)
	}
	if _, err := provider.Get(context.Background(), key+"_MISSING"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing Get error = %v, want ErrNotFound", err)
	}
}

func TestFileProviderReadsOnlyHardenedRegularFiles(t *testing.T) {
	dir := t.TempDir()
	provider := FileProvider{Dir: dir}
	if _, err := provider.Get(context.Background(), "../JWT_SECRET"); err == nil {
		t.Fatal("path traversal key was accepted")
	}
	if _, err := provider.Get(context.Background(), "jwt_secret"); err == nil {
		t.Fatal("lowercase key was accepted")
	}
	if _, err := provider.Get(context.Background(), "MISSING"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing secret error = %v", err)
	}

	secretPath := filepath.Join(dir, "JWT_SECRET")
	if err := os.WriteFile(secretPath, []byte("  hardened-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := provider.Get(context.Background(), "JWT_SECRET")
	if err != nil || got != "hardened-secret" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if info := provider.Info(); info.Name != "file" || info.External || info.Location != dir {
		t.Fatalf("Info = %#v", info)
	}
	if health := provider.Health(context.Background()); !health.Healthy {
		t.Fatalf("Health = %#v", health)
	}

	if err := os.WriteFile(filepath.Join(dir, "EMPTY"), []byte(" \n\t"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Get(context.Background(), "EMPTY"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty file error = %v, want ErrNotFound", err)
	}
}

func TestFileProviderRejectsSymlinkDirectoryAndUnavailableHealth(t *testing.T) {
	dir := t.TempDir()
	provider := FileProvider{Dir: dir}
	target := filepath.Join(dir, "TARGET")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "LINK")); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Get(context.Background(), "LINK"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "DIRECTORY"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Get(context.Background(), "DIRECTORY"); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("directory error = %v", err)
	}

	missing := FileProvider{Dir: filepath.Join(dir, "missing")}.Health(context.Background())
	if missing.Healthy || missing.Message == "" {
		t.Fatalf("missing dir Health = %#v", missing)
	}
	fileDir := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(fileDir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if health := (FileProvider{Dir: fileDir}).Health(context.Background()); health.Healthy {
		t.Fatalf("file path reported healthy: %#v", health)
	}
}

func TestValidKeyPolicy(t *testing.T) {
	for _, key := range []string{"JWT_SECRET", "A", "ABC_123", "0"} {
		if !validKey(key) {
			t.Errorf("validKey(%q) = false", key)
		}
	}
	for _, key := range []string{"", "lower", "A-B", "A/B", "A.B", "A B", "Å"} {
		if validKey(key) {
			t.Errorf("validKey(%q) = true", key)
		}
	}
}

func TestRemoteProviderStatusAndPayloadFailures(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		wantNotFound bool
		wantText     string
	}{
		{name: "not found", status: http.StatusNotFound, wantNotFound: true},
		{name: "upstream error", status: http.StatusBadGateway, body: strings.Repeat("x", 1024), wantText: "502"},
		{name: "invalid json", status: http.StatusOK, body: "{", wantText: "decode response"},
		{name: "empty value", status: http.StatusOK, body: `{"value":""}`, wantNotFound: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			provider := RemoteProvider{Endpoint: srv.URL, Client: srv.Client()}
			_, err := provider.Get(context.Background(), "JWT_SECRET")
			if tc.wantNotFound {
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("Get error = %v, want ErrNotFound", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("Get error = %v, want containing %q", err, tc.wantText)
			}
		})
	}
}

func TestRemoteProviderAuthorizationInfoAndHealthFailures(t *testing.T) {
	var sawSecretAuth, sawHealthAuth bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer bootstrap-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/secrets/JWT_SECRET":
			sawSecretAuth = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":"managed"}`))
		case "/healthz":
			sawHealthAuth = true
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	provider := RemoteProvider{Endpoint: srv.URL + "/", Token: "bootstrap-token", Client: srv.Client()}
	got, err := provider.Get(context.Background(), "JWT_SECRET")
	if err != nil || got != "managed" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if !sawSecretAuth {
		t.Fatal("secret request did not carry bearer token")
	}
	if info := provider.Info(); info.Name != "remote" || !info.External || info.Location != srv.URL+"/" {
		t.Fatalf("Info = %#v", info)
	}
	health := provider.Health(context.Background())
	if health.Healthy || health.Message != "remote provider health check failed" || !sawHealthAuth {
		t.Fatalf("Health = %#v, sawHealthAuth=%v", health, sawHealthAuth)
	}
}

func TestRemoteProviderValidationAndNetworkFailures(t *testing.T) {
	for _, endpoint := range []string{"", "http://example.test", "://bad", "https:///missing-host"} {
		provider := RemoteProvider{Endpoint: endpoint}
		if _, err := provider.Get(context.Background(), "JWT_SECRET"); err == nil {
			t.Fatalf("Get accepted endpoint %q", endpoint)
		}
		if health := provider.Health(context.Background()); health.Healthy || health.Message == "" {
			t.Fatalf("Health(%q) = %#v", endpoint, health)
		}
	}

	client := &http.Client{
		Timeout:   100 * time.Millisecond,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	provider := RemoteProvider{Endpoint: "https://127.0.0.1:1", Client: client}
	if _, err := provider.Get(context.Background(), "JWT_SECRET"); err == nil || !strings.Contains(err.Error(), "remote get") {
		t.Fatalf("network Get error = %v", err)
	}
	if health := provider.Health(context.Background()); health.Healthy || health.Message != "remote provider unreachable" {
		t.Fatalf("network Health = %#v", health)
	}

	if _, err := (RemoteProvider{Endpoint: "https://example.test"}).Get(context.Background(), "lowercase"); err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("invalid-key Get error = %v", err)
	}
}

func TestRemoteProviderHealthyResponse(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if health := (RemoteProvider{Endpoint: srv.URL, Client: srv.Client()}).Health(context.Background()); !health.Healthy || health.Message != "" {
		t.Fatalf("Health = %#v", health)
	}
}
