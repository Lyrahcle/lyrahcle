package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerHandlersUseStrictRequestJSONDecoder(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "json.NewDecoder(r.Body).Decode") {
			t.Fatalf("%s bypasses decodeStrictJSON", entry.Name())
		}
	}
}

func TestSecurityCriticalIgnoredErrorsDoNotRegress(t *testing.T) {
	files := map[string][]string{
		"main.go": {"_ = sched.Every", "_ = srv.Close()"},
		filepath.Join("..", "..", "internal", "sessionstore", "sessionstore.go"): {"_, _ = getPipe.Exec"},
		filepath.Join("..", "..", "internal", "store", "agent_keys.go"):          {"_, _ = s.pool.Exec"},
		filepath.Join("..", "..", "internal", "store", "agent_identity.go"):      {"_, _ = s.pool.Exec"},
		"cross_domain.go":     {"recentAudit, _ =", "contextualAlerts, _ =", "operationalEvents, _ =", "_, _ = incidents.AddEvidence"},
		"orbital_security.go": {"_, _ = incidents.AddEvidence"},
	}
	for name, forbidden := range files {
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		for _, pattern := range forbidden {
			if strings.Contains(text, pattern) {
				t.Fatalf("%s still contains ignored security error pattern %q", name, pattern)
			}
		}
	}
}
