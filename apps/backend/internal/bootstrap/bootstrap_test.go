package bootstrap

import (
	"net/http"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
)

func testConfig() config.Config {
	return config.Config{
		Port:                              "8080",
		WebAuthnRPID:                      "localhost",
		WebAuthnOrigins:                   []string{"http://localhost:5173"},
		RateLimitPlatformCeilingPerMinute: 50000,
		RateLimitTenantReadPerMinute:      1200,
		RateLimitActorReadPerMinute:       300,
		RateLimitTenantWritePerMinute:     300,
		RateLimitActorWritePerMinute:      120,
		RateLimitTenantAgentPerMinute:     12000,
		RateLimitAgentPerMinute:           3000,
	}
}

func TestNewStoresBuildsAllRepositories(t *testing.T) {
	stores := NewStores(nil)
	if stores.Assets == nil || stores.RBAC == nil || stores.AgentIdentities == nil || stores.OrbitalEvents == nil || stores.TenantScope == nil {
		t.Fatal("expected bootstrap to construct all repository handles")
	}
}

func TestOptionalSecurityBuildersRemainDisabledWithoutConfiguration(t *testing.T) {
	cfg := testConfig()
	signer, provider, err := BuildAgentConfigSigner(cfg)
	if err != nil {
		t.Fatalf("BuildAgentConfigSigner returned error: %v", err)
	}
	if signer != nil || provider != nil {
		t.Fatal("expected agent config signing to remain disabled")
	}
	ca, err := BuildAgentCA(cfg)
	if err != nil {
		t.Fatalf("BuildAgentCA returned error: %v", err)
	}
	if ca != nil {
		t.Fatal("expected agent CA to remain disabled")
	}
}

func TestBuildRateLimitersWithoutRedis(t *testing.T) {
	limiters, err := BuildRateLimiters(testConfig(), nil)
	if err != nil {
		t.Fatalf("BuildRateLimiters returned error: %v", err)
	}
	if limiters.Global == nil || limiters.Login == nil || limiters.Tenant == nil {
		t.Fatal("expected all rate limiters")
	}
}

func TestBuildWebAuthnHandler(t *testing.T) {
	handler, err := BuildWebAuthnHandler(testConfig())
	if err != nil {
		t.Fatalf("BuildWebAuthnHandler returned error: %v", err)
	}
	if handler == nil {
		t.Fatal("expected WebAuthn handler")
	}
}

func TestNewHTTPServerKeepsOperationalTimeouts(t *testing.T) {
	server := NewHTTPServer(testConfig(), http.NewServeMux())
	if server.Addr != ":8080" {
		t.Fatalf("unexpected address: %s", server.Addr)
	}
	if server.ReadTimeout != 15*time.Second || server.WriteTimeout != 30*time.Second || server.IdleTimeout != 60*time.Second {
		t.Fatal("unexpected HTTP server timeouts")
	}
}

func TestConfigureAgentClientTLSNoopsWithoutCA(t *testing.T) {
	server := &http.Server{}
	if err := ConfigureAgentClientTLS(server, testConfig()); err != nil {
		t.Fatalf("ConfigureAgentClientTLS returned error: %v", err)
	}
	if server.TLSConfig != nil {
		t.Fatal("expected TLS client configuration to remain unset")
	}
}
