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
	agentSrc := readRouteSource(t, "agent_identity.go")
	rbacSrc := readRouteSource(t, "rbac.go")
	tenantScopeSrc := readRouteSource(t, "tenant_scope.go")

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
		"agent_identity.go": {`"agents.read"`, `"agents.manage"`},
		"rbac.go":          {`"rbac.manage"`},
		"tenant_scope.go":  {`"organizations.read"`},
	}
	sources := map[string]string{
		"server route package":  serverSrc,
		"agent_identity.go": agentSrc,
		"rbac.go":          rbacSrc,
		"tenant_scope.go":  tenantScopeSrc,
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
	agentSrc := readRouteSource(t, "agent_identity.go")
	orgSrc := readRouteSource(t, "organizations.go")

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
