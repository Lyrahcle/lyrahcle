package config

import "testing"

func TestValidateProductionRedisURLRequiresTLSAndAuth(t *testing.T) {
	invalid := []string{
		"redis://default:secret@example.cache.amazonaws.com:6379/0",
		"rediss://example.cache.amazonaws.com:6379/0",
		"rediss://default@example.cache.amazonaws.com:6379/0",
	}
	for _, target := range invalid {
		if err := validateProductionRedisURL(target); err == nil {
			t.Fatalf("expected rejection for %q", target)
		}
	}
	if err := validateProductionRedisURL("rediss://default:strong-token@example.cache.amazonaws.com:6379/0"); err != nil {
		t.Fatalf("valid production Redis URL rejected: %v", err)
	}
}
