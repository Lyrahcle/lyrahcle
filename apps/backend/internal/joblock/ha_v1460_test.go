package joblock

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestTryRejectsInvalidLeaseInputs(t *testing.T) {
	locker := New(nil)
	if _, acquired, err := locker.Try(context.Background(), "", time.Second); err == nil || acquired {
		t.Fatalf("empty job: acquired=%v err=%v", acquired, err)
	}
	for _, ttl := range []time.Duration{0, -time.Second} {
		if _, acquired, err := locker.Try(context.Background(), "job", ttl); err == nil || acquired {
			t.Fatalf("ttl %s: acquired=%v err=%v", ttl, acquired, err)
		}
	}
}

func TestLeaseHeartbeatPreventsDuplicateExecutionAfterOriginalTTL(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	job := "heartbeat-" + uuid.NewString()

	first := New(client)
	second := New(client)
	unlock, acquired, err := first.Try(ctx, job, 90*time.Millisecond)
	if err != nil || !acquired {
		t.Fatalf("first Try: acquired=%v err=%v", acquired, err)
	}

	time.Sleep(230 * time.Millisecond)
	if _, acquired, err := second.Try(ctx, job, time.Second); err != nil || acquired {
		t.Fatalf("second Try after original TTL: acquired=%v err=%v; heartbeat must retain ownership", acquired, err)
	}

	if err := unlock(ctx); err != nil {
		t.Fatal(err)
	}
	unlockSecond, acquired, err := second.Try(ctx, job, time.Second)
	if err != nil || !acquired {
		t.Fatalf("second Try after release: acquired=%v err=%v", acquired, err)
	}
	if err := unlockSecond(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestHeartbeatStopsWhenLeaseOwnershipIsLost(t *testing.T) {
	client := testRedis(t)
	ctx := context.Background()
	job := "ownership-loss-" + uuid.NewString()
	locker := New(client)

	unlock, acquired, err := locker.Try(ctx, job, 120*time.Millisecond)
	if err != nil || !acquired {
		t.Fatalf("Try: acquired=%v err=%v", acquired, err)
	}

	key := defaultPrefix + job
	if err := client.Set(ctx, key, "replacement-owner", time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(70 * time.Millisecond)

	if got, err := client.Get(ctx, key).Result(); err != nil || got != "replacement-owner" {
		t.Fatalf("replacement lease changed: value=%q err=%v", got, err)
	}
	// Stale unlock remains token-guarded and cannot remove the replacement.
	if err := unlock(ctx); err != nil {
		t.Fatal(err)
	}
	if exists, err := client.Exists(ctx, key).Result(); err != nil || exists != 1 {
		t.Fatalf("replacement lease removed: exists=%d err=%v", exists, err)
	}
}

func TestAcquireRedisFailureIsReturned(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:       "unavailable.test:6379",
		MaxRetries: 0,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("simulated redis outage")
		},
	})
	t.Cleanup(func() { _ = client.Close() })

	if _, acquired, err := New(client).Try(context.Background(), "job", time.Second); err == nil || acquired {
		t.Fatalf("acquired=%v err=%v want false,error", acquired, err)
	}
}
