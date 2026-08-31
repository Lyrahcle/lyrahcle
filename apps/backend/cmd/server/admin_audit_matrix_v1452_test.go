package main

import (
	"os"
	"strings"
	"testing"
)

func requireAuditContractTokens(t *testing.T, filename string, tokens ...string) {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range tokens {
		if !strings.Contains(src, token) {
			t.Fatalf("%s administrative audit contract missing %q", filename, token)
		}
	}
}

func TestAdministrativeAuditMatrixControlPlane(t *testing.T) {
	requireAuditContractTokens(t, "organizations_v100.go",
		"audit.ActionOrganizationCreated",
		"audit.ActionOrganizationUpdated",
		"audit.ActionPlatformAdminGranted",
		"audit.ActionPlatformAdminRevoked",
		"logAuditInOrganization",
		`"actor_organization_id"`,
		`"target_user_id"`,
	)
}

func TestAdministrativeAuditMatrixSigningKeys(t *testing.T) {
	requireAuditContractTokens(t, "signing_keys_v136.go",
		"audit.ActionSigningKeyPolicyUpdated",
		"audit.ActionSigningKeyPrepared",
		"audit.ActionSigningKeyActivated",
		"audit.ActionSigningKeyRevoked",
		"logSigningKeyAudit",
		"logAuditInOrganization",
	)
}

func TestAdministrativeAuditMatrixAgentIdentity(t *testing.T) {
	requireAuditContractTokens(t, "agent_identity_v90.go",
		"audit.ActionAgentEnrollmentIssued",
		"audit.ActionAgentStationUpdated",
		"audit.ActionAgentStatusChanged",
		`"agent_id"`,
	)
}

func TestAdministrativeAuditMatrixAgentConfig(t *testing.T) {
	requireAuditContractTokens(t, "agent_config_v91.go",
		"audit.ActionAgentConfigIssued",
		"audit.ActionAgentConfigRollback",
		"logAudit",
		`"agent_id"`,
		`"version"`,
		`"signing_kid"`,
	)
}

func TestAdministrativeAuditMatrixUsesCentralActionCatalog(t *testing.T) {
	requireAuditContractTokens(t, "../../internal/audit/audit.go",
		`ActionOrganizationCreated        = "organization.created"`,
		`ActionOrganizationUpdated        = "organization.updated"`,
		`ActionPlatformAdminGranted       = "platform_admin.granted"`,
		`ActionPlatformAdminRevoked       = "platform_admin.revoked"`,
		`ActionSigningKeyPolicyUpdated    = "signing_key.policy_updated"`,
		`ActionAgentEnrollmentIssued      = "agent.enrollment_issued"`,
		`ActionAgentStationUpdated        = "agent.station_updated"`,
		`ActionAgentStatusChanged         = "agent.status_changed"`,
		`ActionAgentConfigIssued          = "agent.config_issued"`,
		`ActionAgentConfigRollback        = "agent.config_rollback"`,
	)
}
