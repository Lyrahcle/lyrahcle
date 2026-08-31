package main

import (
	"os"
	"strings"
	"testing"
)

func readRouteSource(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRBACRouteMatrixCoversSensitiveDomains(t *testing.T) {
	serverSrc := readServerProductionSource(t)
	agentSrc := readRouteSource(t, "agent_identity_v90.go")
	rbacSrc := readRouteSource(t, "rbac_v110.go")
	tenantScopeSrc := readRouteSource(t, "tenant_scope_v102.go")

	required := map[string][]string{
		"server route package": {
			`"users.read"`, `"users.manage"`,
			`"assets.read"`, `"assets.manage"`,
			`"programs.read"`, `"programs.manage"`,
			`"incidents.read"`, `"incidents.manage"`,
			`"rf.read"`, `"rf.ingest"`,
			`"detections.read"`, `"detections.manage"`,
			`"alerts.read"`, `"alerts.manage"`,
			`"audit.read"`,
			`"agents.read"`, `"agents.manage"`,
		},
		"agent_identity_v90.go": {`"agents.read"`, `"agents.manage"`},
		"rbac_v110.go":          {`"rbac.manage"`},
		"tenant_scope_v102.go":  {`"organizations.read"`},
	}
	sources := map[string]string{
		"server route package":  serverSrc,
		"agent_identity_v90.go": agentSrc,
		"rbac_v110.go":          rbacSrc,
		"tenant_scope_v102.go":  tenantScopeSrc,
	}
	for file, tokens := range required {
		for _, token := range tokens {
			if !strings.Contains(sources[file], token) {
				t.Errorf("%s missing RBAC capability %s", file, token)
			}
		}
	}

	agentKeySrc := readRouteSource(t, "routes_agents.go")
	if strings.Contains(agentKeySrc, `"incidents.manage"`) {
		t.Fatal("agent-key administration must not require incidents.manage")
	}
}

func TestAgentAndOrganizationAdminRoutesDoNotDependOnLegacyAdminRole(t *testing.T) {
	agentSrc := readRouteSource(t, "agent_identity_v90.go")
	orgSrc := readRouteSource(t, "organizations_v100.go")

	legacyAdminGate := `RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin`
	if strings.Contains(agentSrc, legacyAdminGate) {
		t.Error("agent identity administration still depends on legacy admin role")
	}
	if strings.Contains(orgSrc, legacyAdminGate) {
		t.Error("organization administration still depends on legacy admin role")
	}
	for _, token := range []string{"IsPlatformAdministrator", "requirePlatformAdmin"} {
		if !strings.Contains(orgSrc, token) {
			t.Errorf("organization control-plane route missing platform boundary %q", token)
		}
	}
	if strings.Contains(orgSrc, `"organizations.read"`) || strings.Contains(orgSrc, `"organizations.manage"`) {
		t.Error("global organization routes must not use tenant-local organization capabilities")
	}
}
