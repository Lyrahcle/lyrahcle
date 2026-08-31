package store

import (
	"os"
	"strings"
	"testing"
)

func TestAgentConfigSigningKIDIsPersistedWithImmutableVersion(t *testing.T) {
	source, err := os.ReadFile("agent_config.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"SigningKID string",
		"checksum_sha256, signing_kid, created_by",
		"signed.SigningKID",
		"v.checksum_sha256, v.signing_kid",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("agent config signing KID persistence contract missing %q", required)
		}
	}

	migration, err := os.ReadFile("../../migrations/0036_agent_config_signing_kid.sql")
	if err != nil {
		t.Fatal(err)
	}
	migrationText := string(migration)
	for _, required := range []string{
		"ADD COLUMN IF NOT EXISTS signing_kid",
		"signing_kid = ''",
		"ALTER COLUMN signing_kid DROP DEFAULT",
	} {
		if !strings.Contains(migrationText, required) {
			t.Fatalf("signing KID migration contract missing %q", required)
		}
	}
}
