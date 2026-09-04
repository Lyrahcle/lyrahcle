package main

import (
	"os"
	"strings"
	"testing"
)

func TestHybridAndOperationalRoutesUseCapabilitiesV121(t *testing.T) {
	src := readServerProductionSource(t)

	if strings.Contains(src, "auth.RoleOperator") {
		t.Fatal("production routes still contain a legacy RoleOperator authorization gate")
	}
	for _, token := range []string{
		`RequireAgentCredentialOrPermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "rf.ingest"`,
		`a.perm("rf.read")`,
		`a.perm("detections.read")`,
		`a.perm("detections.manage")`,
		`a.perm("alerts.read")`,
		`a.perm("alerts.manage")`,
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("server route package missing capability middleware %q", token)
		}
	}

	if !strings.Contains(src, `meAgentAuth := auth.RequireAgentCredential(`) {
		t.Fatal("/agents/me device route must be machine-credential only")
	}
}

func TestCertificateRotationIsMachineCredentialOnlyV121(t *testing.T) {
	data, err := os.ReadFile("agent_identity.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if strings.Contains(src, "auth.RoleOperator") {
		t.Fatal("agent certificate rotation still carries a RoleOperator fallback")
	}
	if !strings.Contains(src, `credentialAuth := auth.RequireAgentCredential(keyLookup, certLookup)`) {
		t.Fatal("agent certificate rotation must use machine credentials directly")
	}
}
