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
	mainSrc := readRouteSource(t, "main.go")
	agentSrc := readRouteSource(t, "agent_identity_v90.go")
	orgSrc := readRouteSource(t, "organizations_v100.go")
	rbacSrc := readRouteSource(t, "rbac_v110.go")

	required := map[string][]string{
		"main.go": {
			`"users.read"`, `"users.manage"`,
			`"assets.read"`, `"assets.manage"`,
			`"programs.read"`, `"programs.manage"`,
			`"incidents.read"`, `"incidents.manage"`,
			`"rf.read"`, `"rf.ingest"`,
			`"audit.read"`,
			`"agents.read"`, `"agents.manage"`,
		},
		"agent_identity_v90.go": {`"agents.read"`, `"agents.manage"`},
		"organizations_v100.go": {`"organizations.read"`, `"organizations.manage"`},
		"rbac_v110.go":          {`"rbac.manage"`},
	}
	sources := map[string]string{
		"main.go":               mainSrc,
		"agent_identity_v90.go": agentSrc,
		"organizations_v100.go": orgSrc,
		"rbac_v110.go":          rbacSrc,
	}
	for file, tokens := range required {
		for _, token := range tokens {
			if !strings.Contains(sources[file], token) {
				t.Errorf("%s missing RBAC capability %s", file, token)
			}
		}
	}

	if strings.Contains(mainSrc, `requireRequestPermission(w, r, rbac, "incidents.manage")`) &&
		strings.Contains(mainSrc, `r.Route("/agent-keys"`) {
		start := strings.Index(mainSrc, `r.Route("/agent-keys"`)
		end := strings.Index(mainSrc[start:], `r.With(meAgentAuth).Post("/agents/me/device"`)
		if end > 0 {
			block := mainSrc[start : start+end]
			if strings.Contains(block, `"incidents.manage"`) {
				t.Fatal("agent-key administration must not require incidents.manage")
			}
		}
	}
}

func TestAgentAndOrganizationAdminRoutesDoNotDependOnLegacyAdminRole(t *testing.T) {
	agentSrc := readRouteSource(t, "agent_identity_v90.go")
	orgSrc := readRouteSource(t, "organizations_v100.go")

	for _, token := range []string{
		`RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin`,
	} {
		if strings.Contains(agentSrc, token) {
			t.Errorf("agent identity administration still depends on legacy admin role")
		}
		if strings.Contains(orgSrc, token) {
			t.Errorf("organization administration still depends on legacy admin role")
		}
	}
}
