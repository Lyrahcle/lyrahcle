package main

import (
	"os"
	"strings"
	"testing"
)

func TestTenantOwnedRoutesAvoidKnownGlobalStoreCalls(t *testing.T) {
	src := readServerProductionSource(t)

	forbidden := []string{
		"users.List(r.Context())",
		"assets.List(r.Context()",
		"programs.List(r.Context())",
		"incidents.List(r.Context()",
		"auditLog.List(r.Context()",
		"rf.Recent(r.Context()",
		"rfAnomalies.Recent(r.Context()",
		"assetAlerts.List(r.Context()",
	}
	for _, token := range forbidden {
		if strings.Contains(src, token) {
			t.Fatalf("known global tenant-owned query still present: %s", token)
		}
	}
}

func TestSearchSeparatesGlobalAndTenantOwnedDomains(t *testing.T) {
	data, err := os.ReadFile("../../internal/store/search.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, token := range []string{
		"FROM satellites",
		"FROM space_cves",
		"FROM assets WHERE organization_id=$2",
		"FROM incidents WHERE organization_id=$2",
	} {
		if !strings.Contains(src, token) {
			t.Fatalf("search tenant/global policy missing %q", token)
		}
	}
}
