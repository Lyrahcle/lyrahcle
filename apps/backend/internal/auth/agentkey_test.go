package auth

import "testing"

func TestGenerateAgentKey(t *testing.T) {
	plaintext, hash, err := GenerateAgentKey()
	if err != nil {
		t.Fatalf("GenerateAgentKey returned error: %v", err)
	}
	if len(plaintext) == 0 {
		t.Fatal("GenerateAgentKey returned an empty plaintext key")
	}
	if plaintext[:len(agentKeyPrefix)] != agentKeyPrefix {
		t.Errorf("plaintext key does not start with %q: %q", agentKeyPrefix, plaintext)
	}
	if hash != HashAgentKey(plaintext) {
		t.Error("returned hash does not match HashAgentKey(plaintext)")
	}
}

func TestGenerateAgentKeyUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		plaintext, _, err := GenerateAgentKey()
		if err != nil {
			t.Fatalf("GenerateAgentKey returned error: %v", err)
		}
		if seen[plaintext] {
			t.Fatalf("GenerateAgentKey produced a duplicate key after %d iterations", i)
		}
		seen[plaintext] = true
	}
}

func TestHashAgentKeyDeterministic(t *testing.T) {
	key := "lyr_agent_deadbeef"
	if HashAgentKey(key) != HashAgentKey(key) {
		t.Error("HashAgentKey should be deterministic for the same input")
	}
}

func TestHashAgentKeyDiffersForDifferentKeys(t *testing.T) {
	if HashAgentKey("lyr_agent_a") == HashAgentKey("lyr_agent_b") {
		t.Error("HashAgentKey should produce different hashes for different keys")
	}
}
