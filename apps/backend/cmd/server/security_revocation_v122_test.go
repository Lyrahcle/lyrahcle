package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/sessionstore"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tokenblocklist"
)

func TestRevokeUserSecurityStateFailsClosedWhenRedisUnavailable(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  25 * time.Millisecond,
		ReadTimeout:  25 * time.Millisecond,
		WriteTimeout: 25 * time.Millisecond,
		MaxRetries:   0,
	})
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	err := revokeUserSecurityState(ctx, "user-test", time.Minute, tokenblocklist.New(client), sessionstore.New(client), client)
	if err == nil {
		t.Fatal("expected revocation failure when Redis is unavailable")
	}
}

func TestRevokeUserSecurityStateWithoutRedisIsNoop(t *testing.T) {
	if err := revokeUserSecurityState(context.Background(), "user-test", time.Minute, tokenblocklist.New(nil), sessionstore.New(nil), nil); err != nil {
		t.Fatalf("revocation without configured Redis should be a no-op, got %v", err)
	}
}

func TestSecurityCriticalRoutesDoNotIgnoreRevocationErrors(t *testing.T) {
	data, err := os.ReadFile("auth_v84.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, forbidden := range []string{"_ = blocklist.RevokeUser", "_ = sessions.DeleteUser"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("security-critical route still ignores revocation error: %s", forbidden)
		}
	}
	for _, required := range []string{
		"auth.ActionSecurityRevocationFailed",
		`"password_reset"`,
		`"webauthn_register"`,
		`"webauthn_remove"`,
		"http.StatusServiceUnavailable",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("security revocation contract missing %q", required)
		}
	}
}
