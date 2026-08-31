package joblock

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set, skipping Redis integration test")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Fatalf("redis ping: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestDistributedLeaseAllowsOnlyOneReplica(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	job := "test-" + uuid.NewString()

	first := New(client)
	second := New(client)
	unlockFirst, acquired, err := first.Try(ctx, job, time.Minute)
	if err != nil || !acquired {
		t.Fatalf("first Try: acquired=%v err=%v", acquired, err)
	}
	if _, acquired, err := second.Try(ctx, job, time.Minute); err != nil || acquired {
		t.Fatalf("second Try while leased: acquired=%v err=%v, want false,nil", acquired, err)
	}
	if err := unlockFirst(ctx); err != nil {
		t.Fatalf("unlock first: %v", err)
	}
	unlockSecond, acquired, err := second.Try(ctx, job, time.Minute)
	if err != nil || !acquired {
		t.Fatalf("second Try after release: acquired=%v err=%v", acquired, err)
	}
	if err := unlockSecond(ctx); err != nil {
		t.Fatalf("unlock second: %v", err)
	}
}

func TestExpiredOwnerCannotDeleteReplacementLease(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	job := "test-expiry-" + uuid.NewString()

	first := New(client)
	second := New(client)
	third := New(client)
	unlockFirst, acquired, err := first.Try(ctx, job, 30*time.Millisecond)
	if err != nil || !acquired {
		t.Fatalf("first Try: acquired=%v err=%v", acquired, err)
	}
	time.Sleep(50 * time.Millisecond)
	unlockSecond, acquired, err := second.Try(ctx, job, time.Minute)
	if err != nil || !acquired {
		t.Fatalf("second Try after expiry: acquired=%v err=%v", acquired, err)
	}
	if err := unlockFirst(ctx); err != nil {
		t.Fatalf("stale owner unlock: %v", err)
	}
	if _, acquired, err := third.Try(ctx, job, time.Minute); err != nil || acquired {
		t.Fatalf("third Try after stale unlock: acquired=%v err=%v, replacement lease must remain", acquired, err)
	}
	if err := unlockSecond(ctx); err != nil {
		t.Fatalf("unlock second: %v", err)
	}
}

func TestNilRedisUsesDevelopmentLocalExecution(t *testing.T) {
	unlock, acquired, err := New(nil).Try(context.Background(), "local", time.Second)
	if err != nil || !acquired {
		t.Fatalf("Try without Redis: acquired=%v err=%v", acquired, err)
	}
	if err := unlock(context.Background()); err != nil {
		t.Fatalf("unlock without Redis: %v", err)
	}
}
