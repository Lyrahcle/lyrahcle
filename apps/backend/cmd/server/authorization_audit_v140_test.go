package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type sensitiveRouteAuthorization struct {
	file       string
	route      string
	guardToken string
	capability string
}

// sensitiveRouteAuthorizationMatrix is the release-gate inventory for tenant-sensitive
// human routes. Adding or moving a sensitive route requires updating this table and the
// endpoint matrix documentation in the same change.
var sensitiveRouteAuthorizationMatrix = []sensitiveRouteAuthorization{
	{"main.go", `Get("/",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.read"`, "assets.read"},
	{"main.go", `Post("/",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.manage"`, "assets.manage"},
	{"main.go", `Put("/{assetID}",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.manage"`, "assets.manage"},
	{"main.go", `Delete("/{assetID}",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.manage"`, "assets.manage"},
	{"main.go", `Put("/{programID}",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "programs.manage"`, "programs.manage"},
	{"main.go", `Delete("/{programID}",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "programs.manage"`, "programs.manage"},
	{"main.go", `Get("/readings",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "rf.read"`, "rf.read"},
	{"main.go", `Get("/anomalies",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "rf.read"`, "rf.read"},
	{"main.go", `Get("/{satelliteID}/passes",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.read"`, "orbital.read"},
	{"main.go", `Get("/{satelliteID}/orbital-events",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.read"`, "orbital.read"},
	{"main.go", `Post("/{satelliteID}/orbital-events",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.manage"`, "orbital.manage"},
	{"main.go", `Put("/{satelliteID}",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "orbital.manage"`, "orbital.manage"},
	{"main.go", `Get("/audit-log",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "audit.read"`, "audit.read"},
	{"main.go", `Get("/{incidentID}",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "incidents.read"`, "incidents.read"},
	{"main.go", `Patch("/{incidentID}",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "incidents.manage"`, "incidents.manage"},
	{"main.go", `Post("/{incidentID}/notes",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "incidents.manage"`, "incidents.manage"},
	{"main.go", `Post("/{incidentID}/evidence",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "incidents.manage"`, "incidents.manage"},
	{"main.go", `Post("/{alertID}/acknowledge",`, `RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "alerts.manage"`, "alerts.manage"},
	{"agent_identity_v90.go", `Get("/agents/",`, `r.With(agentRead).`, "agents.read|agents.manage"},
	{"agent_identity_v90.go", `Patch("/agents/{agentID}/status",`, `r.With(agentManage).`, "agents.manage"},
	{"agent_identity_v90.go", `Put("/agents/{agentID}/station",`, `r.With(agentManage).`, "agents.manage"},
	{"agent_config_v91.go", `Get("/agents/{agentID}/config",`, `r.With(agentRead).`, "agents.read|agents.manage"},
	{"agent_config_v91.go", `Get("/agents/{agentID}/config/history",`, `r.With(agentRead).`, "agents.read|agents.manage"},
	{"agent_config_v91.go", `Put("/agents/{agentID}/config",`, `r.With(agentManage).`, "agents.manage"},
	{"agent_config_v91.go", `Post("/agents/{agentID}/config/rollback",`, `r.With(agentManage).`, "agents.manage"},
	{"fleet_rollouts_v141.go", `Get("/agent-rollouts",`, `r.With(read).`, "agents.read|agents.manage"},
	{"fleet_rollouts_v141.go", `Get("/agent-rollouts/{rolloutID}",`, `r.With(read).`, "agents.read|agents.manage"},
	{"fleet_rollouts_v141.go", `Post("/agent-rollouts",`, `r.With(manage).`, "agents.manage"},
	{"fleet_rollouts_v141.go", `Post("/agent-rollouts/{rolloutID}/start",`, `r.With(manage).`, "agents.manage"},
	{"fleet_rollouts_v141.go", `Post("/agent-rollouts/{rolloutID}/advance",`, `r.With(manage).`, "agents.manage"},
	{"fleet_rollouts_v141.go", `Post("/agent-rollouts/{rolloutID}/pause",`, `r.With(manage).`, "agents.manage"},
	{"fleet_rollouts_v141.go", `Post("/agent-rollouts/{rolloutID}/resume",`, `r.With(manage).`, "agents.manage"},
	{"fleet_rollouts_v141.go", `Post("/agent-rollouts/{rolloutID}/rollback",`, `r.With(manage).`, "agents.manage"},
}

func TestSensitiveRouteAuthorizationInventory(t *testing.T) {
	sources := map[string]string{}
	for _, entry := range sensitiveRouteAuthorizationMatrix {
		src, ok := sources[entry.file]
		if !ok {
			data, err := os.ReadFile(entry.file)
			if err != nil {
				t.Fatal(err)
			}
			src = string(data)
			sources[entry.file] = src
		}

		routeAt := strings.Index(src, entry.route)
		if routeAt < 0 {
			t.Errorf("authorization inventory route missing from %s: %s", entry.file, entry.route)
			continue
		}
		windowStart := routeAt - 500
		if windowStart < 0 {
			windowStart = 0
		}
		window := src[windowStart:routeAt]
		if !strings.Contains(window, entry.guardToken) {
			t.Errorf("%s route %s is not immediately guarded by %s (%s)", entry.file, entry.route, entry.guardToken, entry.capability)
		}
	}
}

func TestNoSensitiveLegacyRoleGates(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := regexp.MustCompile(`RequireRoleWithFallback\([^\n]+auth\.Role(Admin|Operator|Analyst|Auditor)`)
	var hits []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range forbidden.FindAllString(string(data), -1) {
			hits = append(hits, file.Name()+": "+match)
		}
	}
	sort.Strings(hits)
	if len(hits) > 0 {
		t.Fatalf("sensitive route still uses a legacy role gate instead of a capability:\n%s", strings.Join(hits, "\n"))
	}
}

func TestHTTPRouteCountRequiresAuthorizationReview(t *testing.T) {
	// This deliberately counts every literal server route registration, including public
	// health/docs endpoints. Any route addition/removal forces a conscious review of its
	// authentication and authorization boundary instead of silently expanding the API.
	routeRegistration := regexp.MustCompile(`(?m)^\s*r(?:\.With\([^\n]*\))?\.(?:Get|Post|Put|Patch|Delete|Handle)\("`)
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	got := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") || strings.HasSuffix(file.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatal(err)
		}
		got += len(routeRegistration.FindAll(data, -1))
	}
	const reviewedRouteRegistrations = 115
	if got != reviewedRouteRegistrations {
		t.Fatalf("HTTP route inventory changed: got %d registrations, reviewed %d; classify every added/removed route before updating this gate", got, reviewedRouteRegistrations)
	}
}

func TestViewerOnlyAuthorizationBoundariesAreExplicitlyReviewed(t *testing.T) {
	// Viewer-only middleware is intentionally limited to authentication/self-service,
	// stateless decoders, search/system metadata, CVE reads, and control-plane wrappers
	// that perform their own platform-admin check. This count is a tripwire: introducing
	// another viewer-only boundary requires an explicit authorization review.
	files := []string{"main.go", "auth_v84.go", "organizations_v100.go", "rate_limit_status_v138.go", "rbac_v110.go", "signing_keys_v136.go", "secret_status_v143.go"}
	const token = "RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)"
	got := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		got += strings.Count(string(data), token)
	}
	const reviewedViewerBoundaries = 18
	if got != reviewedViewerBoundaries {
		t.Fatalf("viewer-only authorization boundaries changed: got %d, reviewed %d; audit the new/removed boundary and update the authorization matrix", got, reviewedViewerBoundaries)
	}
}
