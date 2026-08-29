package ratelimit

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

func TestIncrementThenGetReflectsCount(t *testing.T) {
	c := NewRedisCounter(testClient(t), "test")
	c.Config(100, time.Minute)

	window := time.Now()
	key := "test-key-" + t.Name()

	for i := 0; i < 3; i++ {
		if err := c.Increment(key, window); err != nil {
			t.Fatalf("Increment: %v", err)
		}
	}

	curr, prev, err := c.Get(key, window, window.Add(-time.Minute))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if curr != 3 {
		t.Errorf("current window count = %d, want 3", curr)
	}
	if prev != 0 {
		t.Errorf("previous window count = %d, want 0 (nothing incremented there)", prev)
	}
}

func TestIncrementByAddsTheGivenAmount(t *testing.T) {
	c := NewRedisCounter(testClient(t), "test")
	c.Config(100, time.Minute)

	window := time.Now()
	key := "test-key-by-" + t.Name()

	if err := c.IncrementBy(key, window, 7); err != nil {
		t.Fatalf("IncrementBy: %v", err)
	}
	if err := c.IncrementBy(key, window, 3); err != nil {
		t.Fatalf("IncrementBy: %v", err)
	}

	curr, _, err := c.Get(key, window, window.Add(-time.Minute))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if curr != 10 {
		t.Errorf("current window count = %d, want 10", curr)
	}
}

func TestGetOnNeverIncrementedKeyIsZero(t *testing.T) {
	c := NewRedisCounter(testClient(t), "test")
	c.Config(100, time.Minute)

	window := time.Now()
	curr, prev, err := c.Get("never-incremented-"+t.Name(), window, window.Add(-time.Minute))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if curr != 0 || prev != 0 {
		t.Errorf("got (%d, %d), want (0, 0)", curr, prev)
	}
}

func TestDifferentWindowsAreIndependent(t *testing.T) {
	c := NewRedisCounter(testClient(t), "test")
	c.Config(100, time.Minute)

	key := "test-key-windows-" + t.Name()
	windowA := time.Now()
	windowB := windowA.Add(time.Hour)

	if err := c.Increment(key, windowA); err != nil {
		t.Fatalf("Increment windowA: %v", err)
	}

	currB, _, err := c.Get(key, windowB, windowB.Add(-time.Minute))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if currB != 0 {
		t.Errorf("windowB count = %d, want 0 (independent from windowA)", currB)
	}
}

func TestNamespacesAreIndependent(t *testing.T) {
	client := testClient(t)
	global := NewRedisCounter(client, "global-test-"+t.Name())
	login := NewRedisCounter(client, "login-test-"+t.Name())
	global.Config(120, time.Minute)
	login.Config(10, time.Minute)

	window := time.Now()
	key := "shared-ip"

	for i := 0; i < 20; i++ {
		if err := global.Increment(key, window); err != nil {
			t.Fatalf("global Increment: %v", err)
		}
	}
	if err := login.Increment(key, window); err != nil {
		t.Fatalf("login Increment: %v", err)
	}

	globalCount, _, err := global.Get(key, window, window.Add(-time.Minute))
	if err != nil {
		t.Fatalf("global Get: %v", err)
	}
	loginCount, _, err := login.Get(key, window, window.Add(-time.Minute))
	if err != nil {
		t.Fatalf("login Get: %v", err)
	}
	if globalCount != 20 {
		t.Fatalf("global count = %d, want 20", globalCount)
	}
	if loginCount != 1 {
		t.Fatalf("login count = %d, want 1", loginCount)
	}
}

func TestBucketKeysIncludeNamespace(t *testing.T) {
	window := time.Unix(1_700_000_000, 0)
	global := NewRedisCounter(nil, "global")
	login := NewRedisCounter(nil, "login")
	globalKey := global.bucketKey("192.0.2.10", window)
	loginKey := login.bucketKey("192.0.2.10", window)
	if globalKey == loginKey {
		t.Fatalf("global and login bucket keys collide: %q", globalKey)
	}
}
