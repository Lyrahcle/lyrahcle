package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlko01/lyrahcle/apps/backend/internal/secretprovider"
)

// loadSecretProvider defaults to the environment-variable provider when
// SECRET_PROVIDER is unset, which is the safe out-of-the-box behavior for
// local development and CI.
func TestLoadSecretProvider_DefaultsToEnv(t *testing.T) {
	t.Setenv("SECRET_PROVIDER", "")

	provider := loadSecretProvider(false)

	if _, ok := provider.(secretprovider.EnvProvider); !ok {
		t.Fatalf("expected EnvProvider by default, got %T", provider)
	}
	if name := provider.Info().Name; name != "env" {
		t.Fatalf("expected provider name %q, got %q", "env", name)
	}
}

// SECRET_PROVIDER=file with SECRET_FILE_DIR set must select a FileProvider
// pointed at that directory, and must not touch the network or crash the
// process before Load() gets a chance to run its own production health gate.
func TestLoadSecretProvider_FileMode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SECRET_PROVIDER", "file")
	t.Setenv("SECRET_FILE_DIR", dir)

	provider := loadSecretProvider(false)

	fileProvider, ok := provider.(secretprovider.FileProvider)
	if !ok {
		t.Fatalf("expected FileProvider, got %T", provider)
	}
	if fileProvider.Dir != dir {
		t.Fatalf("expected FileProvider.Dir %q, got %q", dir, fileProvider.Dir)
	}
	if !provider.Health(context.Background()).Healthy {
		t.Fatalf("expected file provider pointed at an existing directory to be healthy")
	}
}

// SECRET_PROVIDER=remote outside of production must not perform the
// blocking health check that Load() enforces only when production is true;
// it should simply construct the RemoteProvider from the endpoint and an
// inline token.
func TestLoadSecretProvider_RemoteMode_NonProduction_SkipsHealthCheck(t *testing.T) {
	t.Setenv("SECRET_PROVIDER", "remote")
	t.Setenv("SECRET_PROVIDER_ENDPOINT", "https://vault.internal.example")
	t.Setenv("SECRET_PROVIDER_TOKEN", "inline-token")
	t.Setenv("SECRET_PROVIDER_TOKEN_FILE", "")

	provider := loadSecretProvider(false)

	remote, ok := provider.(secretprovider.RemoteProvider)
	if !ok {
		t.Fatalf("expected RemoteProvider, got %T", provider)
	}
	if remote.Endpoint != "https://vault.internal.example" {
		t.Fatalf("unexpected endpoint: %q", remote.Endpoint)
	}
	if remote.Token != "inline-token" {
		t.Fatalf("expected inline token to be used, got %q", remote.Token)
	}
}

// When SECRET_PROVIDER_TOKEN is empty but SECRET_PROVIDER_TOKEN_FILE points
// at a readable file, the token must be read from disk and trimmed.
func TestLoadSecretProvider_RemoteMode_ReadsTokenFromFile(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenPath, []byte("file-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	t.Setenv("SECRET_PROVIDER", "remote")
	t.Setenv("SECRET_PROVIDER_ENDPOINT", "https://vault.internal.example")
	t.Setenv("SECRET_PROVIDER_TOKEN", "")
	t.Setenv("SECRET_PROVIDER_TOKEN_FILE", tokenPath)

	provider := loadSecretProvider(false)

	remote, ok := provider.(secretprovider.RemoteProvider)
	if !ok {
		t.Fatalf("expected RemoteProvider, got %T", provider)
	}
	if remote.Token != "file-token" {
		t.Fatalf("expected token read from file to be trimmed, got %q", remote.Token)
	}
}

// getSecret must fall back to the caller-supplied default when the provider
// reports the key as not found, without treating that as a fatal error.
func TestGetSecret_FallsBackOnNotFound(t *testing.T) {
	got := getSecret(secretprovider.EnvProvider{}, "LYRAHCLE_CONFIG_TEST_MISSING_KEY", "fallback-value")
	if got != "fallback-value" {
		t.Fatalf("expected fallback value, got %q", got)
	}
}

// getSecret must return the provider's value untouched when present, taking
// priority over the fallback.
func TestGetSecret_ReturnsProviderValueWhenPresent(t *testing.T) {
	t.Setenv("LYRAHCLE_CONFIG_TEST_PRESENT_KEY", "provider-value")

	got := getSecret(secretprovider.EnvProvider{}, "LYRAHCLE_CONFIG_TEST_PRESENT_KEY", "fallback-value")
	if got != "provider-value" {
		t.Fatalf("expected provider value to win over fallback, got %q", got)
	}
}
