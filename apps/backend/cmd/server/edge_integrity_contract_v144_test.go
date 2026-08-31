package main

import (
	"os"
	"strings"
	"testing"
)

func TestEdgeIntegrityContractV144(t *testing.T) {
	b, err := os.ReadFile("../../internal/store/agent_keys.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, v := range []string{"IntegrityManifestSHA256", "IntegrityBaselineSHA256", "IntegrityStatus", "applyIntegrityBaseline"} {
		if !strings.Contains(s, v) {
			t.Fatalf("missing %s", v)
		}
	}
	m, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(m), "integrity baseline and status are read-only") {
		t.Fatal("missing read-only integrity validation")
	}
}
