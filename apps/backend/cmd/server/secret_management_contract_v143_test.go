package main

import (
	"os"
	"strings"
	"testing"
)

func TestSecretManagementContractV143(t *testing.T) {
	cfg, err := os.ReadFile("../../internal/config/config.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"SECRET_PROVIDER", "SECRET_FILE_DIR", "SECRET_PROVIDER_ENDPOINT", "SECRET_PROVIDER_TOKEN_FILE", `cfg.SecretProviderName == "env"`} {
		if !strings.Contains(string(cfg), token) {
			t.Fatalf("secret management config missing %q", token)
		}
	}
	route, err := os.ReadFile("secret_status_v143.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{`Get("/platform/secrets"`, "platform administrator required", "WithPlatformControlPlane", "managed_keys"} {
		if !strings.Contains(string(route), token) {
			t.Fatalf("secret status route missing %q", token)
		}
	}
	provider, err := os.ReadFile("../../internal/secretprovider/provider.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"remote endpoint must be absolute HTTPS", "symlink secret files are not allowed", "must not be group/world accessible"} {
		if !strings.Contains(string(provider), token) {
			t.Fatalf("secret provider hardening missing %q", token)
		}
	}
}

func TestPlatformControlPlaneRLSRegressionV143(t *testing.T) {
	for _, file := range []string{"organizations_v100.go", "signing_keys_v136.go", "rate_limit_status_v138.go", "secret_status_v143.go"} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "WithPlatformControlPlane") {
			t.Fatalf("%s does not explicitly switch to platform control-plane DB scope", file)
		}
	}
}
