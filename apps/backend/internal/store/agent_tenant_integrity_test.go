package store

import (
	"os"
	"strings"
	"testing"
)

func TestLegacyTenantOwnedWritesDelegateOrPropagateOrganization(t *testing.T) {
	assets, err := os.ReadFile("assets.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(assets), "return s.UpsertInOrganization(ctx, DefaultOrganizationID") {
		t.Fatal("legacy asset upsert must delegate to the default tenant")
	}

	keys, err := os.ReadFile("agent_keys.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(keys), "return s.CreateInOrganization(ctx, DefaultOrganizationID") {
		t.Fatal("legacy agent-key creation must delegate to the default tenant")
	}

	identity, err := os.ReadFile("agent_identity.go")
	if err != nil {
		t.Fatal(err)
	}
	identitySrc := string(identity)
	for _, token := range []string{
		"CreateEnrollmentTokenInOrganization(ctx, DefaultOrganizationID",
		"t.organization_id, i.status",
		"insertAgentCertificate(ctx, tx, organizationID, cert)",
		"organization_id, agent_id, serial_number",
	} {
		if !strings.Contains(identitySrc, token) {
			t.Fatalf("agent identity tenant propagation missing %q", token)
		}
	}

	config, err := os.ReadFile("agent_config.go")
	if err != nil {
		t.Fatal(err)
	}
	configSrc := string(config)
	for _, token := range []string{
		"SELECT organization_id, status FROM agent_identities",
		"organization_id, agent_id, version",
		"organization_id, agent_id, desired_version",
		"organization_id, agent_id, version, status",
	} {
		if !strings.Contains(configSrc, token) {
			t.Fatalf("agent config tenant propagation missing %q", token)
		}
	}
}
