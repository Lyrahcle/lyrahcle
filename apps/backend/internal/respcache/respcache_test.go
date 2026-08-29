package respcache

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

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestSetThenGetRoundTrips(t *testing.T) {
	c := New(testClient(t))
	ctx := context.Background()
	key := "test:respcache:" + t.Name()

	c.Set(ctx, key, sample{Name: "ISS", Count: 3}, time.Minute)

	var got sample
	hit, err := c.Get(ctx, key, &got)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected a cache hit after Set")
	}
	if got.Name != "ISS" || got.Count != 3 {
		t.Errorf("got %+v, want {ISS 3}", got)
	}
}

func TestGetMissOnUnknownKey(t *testing.T) {
	c := New(testClient(t))
	ctx := context.Background()

	var got sample
	hit, err := c.Get(ctx, "test:respcache:never-set-"+t.Name(), &got)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hit {
		t.Fatal("expected a cache miss for a key that was never set")
	}
}

func TestSetExpiresWithTTL(t *testing.T) {
	c := New(testClient(t))
	ctx := context.Background()
	key := "test:respcache:ttl:" + t.Name()

	c.Set(ctx, key, sample{Name: "x"}, 50*time.Millisecond)

	var got sample
	hit, _ := c.Get(ctx, key, &got)
	if !hit {
		t.Fatal("expected a cache hit immediately after Set")
	}

	time.Sleep(150 * time.Millisecond)

	hit, _ = c.Get(ctx, key, &got)
	if hit {
		t.Fatal("expected the cached value to have expired by now")
	}
}

func TestNilCacheIsSafe(t *testing.T) {
	var c *Cache
	ctx := context.Background()

	c.Set(ctx, "key", sample{Name: "x"}, time.Minute)

	var got sample
	hit, err := c.Get(ctx, "key", &got)
	if err != nil {
		t.Fatalf("Get on a nil *Cache should not error, got: %v", err)
	}
	if hit {
		t.Fatal("Get on a nil *Cache should always report a miss")
	}
}

func TestNewWithNilClientIsSafe(t *testing.T) {
	c := New(nil)
	ctx := context.Background()

	c.Set(ctx, "key", sample{Name: "x"}, time.Minute)

	var got sample
	hit, err := c.Get(ctx, "key", &got)
	if err != nil {
		t.Fatalf("Get with a nil client should not error, got: %v", err)
	}
	if hit {
		t.Fatal("Get with a nil client should always report a miss")
	}
}
