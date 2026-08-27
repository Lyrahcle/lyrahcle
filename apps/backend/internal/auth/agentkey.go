package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const agentKeyPrefix = "lyr_agent_"

func GenerateAgentKey() (plaintext string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("auth: generate agent key: %w", err)
	}
	plaintext = agentKeyPrefix + hex.EncodeToString(raw)
	hash = HashAgentKey(plaintext)
	return plaintext, hash, nil
}

func HashAgentKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
