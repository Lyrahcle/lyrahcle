package tokenblocklist

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func testClient(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set, skipping Redis integration test")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("connect to test redis at %s: %v", addr, err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestRevokeThenIsRevoked(t *testing.T) {
	b := New(testClient(t))
	ctx := context.Background()
	jti := "test-jti-revoke-" + t.Name()

	if b.IsRevoked(ctx, jti) {
		t.Fatal("a jti that was never revoked reported as revoked")
	}

	if err := b.Revoke(ctx, jti, time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if !b.IsRevoked(ctx, jti) {
		t.Fatal("expected the jti to be revoked after Revoke")
	}
}

func TestRevokeExpiresWithTTL(t *testing.T) {
	b := New(testClient(t))
	ctx := context.Background()
	jti := "test-jti-ttl-" + t.Name()

	if err := b.Revoke(ctx, jti, 50*time.Millisecond); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !b.IsRevoked(ctx, jti) {
		t.Fatal("expected the jti to be revoked immediately after Revoke")
	}

	time.Sleep(150 * time.Millisecond)

	if b.IsRevoked(ctx, jti) {
		t.Fatal("expected the revocation to have expired by now")
	}
}

func TestRevokeEmptyJTIIsNoop(t *testing.T) {
	b := New(testClient(t))
	ctx := context.Background()

	if err := b.Revoke(ctx, "", time.Minute); err != nil {
		t.Fatalf("Revoke with empty jti should be a no-op, got error: %v", err)
	}
	if b.IsRevoked(ctx, "") {
		t.Fatal("an empty jti should never report as revoked")
	}
}

func TestNilBlocklistIsSafe(t *testing.T) {
	var b *Blocklist
	ctx := context.Background()

	if err := b.Revoke(ctx, "some-jti", time.Minute); err != nil {
		t.Fatalf("Revoke on a nil *Blocklist should be a no-op, got error: %v", err)
	}
	if b.IsRevoked(ctx, "some-jti") {
		t.Fatal("IsRevoked on a nil *Blocklist should always return false")
	}
}

func TestNewWithNilClientIsSafe(t *testing.T) {
	b := New(nil)
	ctx := context.Background()

	if err := b.Revoke(ctx, "some-jti", time.Minute); err != nil {
		t.Fatalf("Revoke with a nil client should be a no-op, got error: %v", err)
	}
	if b.IsRevoked(ctx, "some-jti") {
		t.Fatal("IsRevoked with a nil client should always return false")
	}
}
