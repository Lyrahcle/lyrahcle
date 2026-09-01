package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlko01/lyrahcle/apps/backend/internal/secretprovider"
)

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

func TestGetSecret_FallsBackOnNotFound(t *testing.T) {
	got := getSecret(secretprovider.EnvProvider{}, "LYRAHCLE_CONFIG_TEST_MISSING_KEY", "fallback-value")
	if got != "fallback-value" {
		t.Fatalf("expected fallback value, got %q", got)
	}
}

func TestGetSecret_ReturnsProviderValueWhenPresent(t *testing.T) {
	t.Setenv("LYRAHCLE_CONFIG_TEST_PRESENT_KEY", "provider-value")

	got := getSecret(secretprovider.EnvProvider{}, "LYRAHCLE_CONFIG_TEST_PRESENT_KEY", "fallback-value")
	if got != "provider-value" {
		t.Fatalf("expected provider value to win over fallback, got %q", got)
	}
}
