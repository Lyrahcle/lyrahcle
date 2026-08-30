package main

import (
	"os"
	"strings"
	"testing"
)

func routeBlockV120(t *testing.T, src, start, end string) string {
	t.Helper()
	from := strings.Index(src, start)
	if from < 0 {
		t.Fatalf("route block start missing %q", start)
	}
	to := strings.Index(src[from+len(start):], end)
	if to < 0 {
		t.Fatalf("route block end missing %q", end)
	}
	return src[from : from+len(start)+to]
}

func TestOperationalRoutesUseCapabilitiesWithoutLegacyRoleGatesV120(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)

	assets := routeBlockV120(t, src, `r.Route("/assets"`, `r.Route("/programs"`)
	for _, capability := range []string{`"assets.read"`, `"assets.manage"`} {
		if !strings.Contains(assets, capability) {
			t.Fatalf("assets route missing %s", capability)
		}
	}
	if strings.Contains(assets, `"orbital.read"`) || strings.Contains(assets, `"orbital.manage"`) {
		t.Fatal("assets routes must not depend on orbital capabilities")
	}

	programs := routeBlockV120(t, src, `r.Route("/programs"`, `r.Route("/auth"`)
	for _, capability := range []string{`"programs.read"`, `"programs.manage"`} {
		if !strings.Contains(programs, capability) {
			t.Fatalf("program routes missing %s", capability)
		}
	}
	if strings.Contains(programs, `auth.RoleOperator`) || strings.Contains(programs, `auth.RoleAdmin`) {
		t.Fatal("program routes still depend on a legacy role gate")
	}

	satellites := routeBlockV120(t, src, `r.Route("/satellites"`, `r.Route("/tmtc"`)
	for _, capability := range []string{`"orbital.read"`, `"orbital.manage"`} {
		if !strings.Contains(satellites, capability) {
			t.Fatalf("satellite routes missing %s", capability)
		}
	}
	if strings.Contains(satellites, `auth.RoleOperator`) || strings.Contains(satellites, `auth.RoleAdmin`) {
		t.Fatal("satellite routes still depend on a legacy role gate")
	}

	incidents := routeBlockV120(t, src, `r.Route("/incidents"`, `r.Route("/asset-alerts"`)
	for _, capability := range []string{`"incidents.read"`, `"incidents.manage"`} {
		if !strings.Contains(incidents, capability) {
			t.Fatalf("incident routes missing %s", capability)
		}
	}
	for _, legacy := range []string{`auth.RoleOperator`, `auth.RoleAnalyst`, `auth.RoleAdmin`, `only admins can assign incidents`} {
		if strings.Contains(incidents, legacy) {
			t.Fatalf("incident routes still contain legacy authorization %q", legacy)
		}
	}

	tenantScope, err := os.ReadFile("tenant_scope_v102.go")
	if err != nil {
		t.Fatal(err)
	}
	tenantScopeSrc := string(tenantScope)
	if !strings.Contains(tenantScopeSrc, `"organizations.read"`) || strings.Contains(tenantScopeSrc, `auth.RoleAdmin`) {
		t.Fatal("tenant organization scope must use organizations.read without a legacy admin gate")
	}

	if !strings.Contains(src, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "audit.read", isTokenRevoked)).Get("/audit-log"`) {
		t.Fatal("audit log must be authorized directly by audit.read")
	}
}

func TestAgentReadSurfacesAllowReadOrManageCapabilityV120(t *testing.T) {
	for _, file := range []string{"main.go", "agent_identity_v90.go", "agent_config_v91.go"} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		if !strings.Contains(src, `RequireAnyPermissionWithFallback`) || !strings.Contains(src, `[]string{"agents.read", "agents.manage"}`) {
			t.Fatalf("%s does not allow agents.manage to access required read surfaces", file)
		}
	}
}
