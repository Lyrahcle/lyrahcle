package ratelimit

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func unavailableRedisV1457(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:       "unavailable.test:6379",
		MaxRetries: 0,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("simulated redis outage")
		},
	})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestPolicyValidationRejectsUnsafeBudgets(t *testing.T) {
	base := testPolicy()
	tests := []Policy{
		func() Policy { p := base; p.Window = 500 * time.Millisecond; return p }(),
		func() Policy { p := base; p.Window = 2 * time.Hour; return p }(),
		func() Policy { p := base; p.TenantReadLimit = 0; return p }(),
		func() Policy { p := base; p.ActorReadLimit = p.TenantReadLimit + 1; return p }(),
		func() Policy { p := base; p.ActorWriteLimit = p.TenantWriteLimit + 1; return p }(),
		func() Policy { p := base; p.AgentLimit = p.TenantAgentLimit + 1; return p }(),
	}
	for i, p := range tests {
		if err := p.Validate(); err == nil {
			t.Fatalf("case %d unexpectedly valid: %+v", i, p)
		}
	}
}

func TestHumanInputValidationAndUnsupportedClass(t *testing.T) {
	l, err := NewTenantLimiter(nil, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	org, user := uuid.New(), uuid.New()
	if _, err := l.AllowHuman(context.Background(), uuid.Nil, user, ClassRead); err == nil {
		t.Fatal("nil org must fail")
	}
	if _, err := l.AllowHuman(context.Background(), org, uuid.Nil, ClassRead); err == nil {
		t.Fatal("nil user must fail")
	}
	if _, err := l.AllowHuman(context.Background(), org, user, "admin"); err == nil {
		t.Fatal("unsupported class must fail")
	}
	if _, err := l.AllowAgent(context.Background(), uuid.Nil, "agent"); err == nil {
		t.Fatal("nil agent org must fail")
	}
	if _, err := l.AllowAgent(context.Background(), org, ""); err == nil {
		t.Fatal("empty agent id must fail")
	}
}

func TestTenantBucketKeyIsStableAndPrivacyPreserving(t *testing.T) {
	when := time.Unix(1700000000, 0).UTC()
	a := tenantBucketKey("actor", ClassRead, "tenant:user", when)
	b := tenantBucketKey("actor", ClassRead, "tenant:user", when)
	c := tenantBucketKey("actor", ClassRead, "tenant:user2", when)
	if a != b {
		t.Fatal("same input must produce stable bucket key")
	}
	if a == c {
		t.Fatal("different subjects must not collide in test fixture")
	}
	if contains(a, "tenant:user") || contains(a, "tenant:user2") {
		t.Fatalf("bucket key leaks raw subject: %s", a)
	}
}

func TestRedisFailureFallsBackToBoundedLocalLimiter(t *testing.T) {
	p := testPolicy()
	p.TenantReadLimit = 2
	p.ActorReadLimit = 2
	l, err := NewTenantLimiter(unavailableRedisV1457(t), p)
	if err != nil {
		t.Fatal(err)
	}
	fixedNow := time.Unix(1700000000, 0).UTC()
	l.now = func() time.Time { return fixedNow }

	org, user := uuid.New(), uuid.New()
	for i := 0; i < 2; i++ {
		d, err := l.AllowHuman(context.Background(), org, user, ClassRead)
		if err != nil {
			t.Fatal(err)
		}
		if !d.Allowed {
			t.Fatalf("request %d unexpectedly blocked: %+v", i+1, d)
		}
	}
	d, err := l.AllowHuman(context.Background(), org, user, ClassRead)
	if err != nil {
		t.Fatal(err)
	}
	if d.Allowed || d.Remaining != 0 || d.Scope != "tenant" {
		t.Fatalf("fallback decision=%+v want blocked tenant budget", d)
	}
}

func TestRedisSharedLimiterSeparatesTenantsAndActors(t *testing.T) {
	addr := testRedisAddr(t)
	client1 := redis.NewClient(&redis.Options{Addr: addr})
	client2 := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client1.Close(); _ = client2.Close() })

	p := testPolicy()
	p.Window = time.Minute
	p.TenantReadLimit = 2
	p.ActorReadLimit = 2
	first, err := NewTenantLimiter(client1, p)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewTenantLimiter(client2, p)
	if err != nil {
		t.Fatal(err)
	}
	orgA, orgB, user := uuid.New(), uuid.New(), uuid.New()

	if d, err := first.AllowHuman(context.Background(), orgA, user, ClassRead); err != nil || !d.Allowed {
		t.Fatalf("first=%+v err=%v", d, err)
	}
	if d, err := second.AllowHuman(context.Background(), orgA, user, ClassRead); err != nil || !d.Allowed {
		t.Fatalf("second=%+v err=%v", d, err)
	}
	if d, err := first.AllowHuman(context.Background(), orgA, user, ClassRead); err != nil || d.Allowed {
		t.Fatalf("third=%+v err=%v want blocked", d, err)
	}

	// Same user UUID in another organization must receive an independent budget.
	if d, err := second.AllowHuman(context.Background(), orgB, user, ClassRead); err != nil || !d.Allowed {
		t.Fatalf("other tenant=%+v err=%v want allowed", d, err)
	}
}

func testRedisAddr(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	return addr
}
