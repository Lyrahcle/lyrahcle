package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testPolicy() Policy {
	return Policy{
		Window:           time.Minute,
		TenantReadLimit:  4,
		ActorReadLimit:   2,
		TenantWriteLimit: 2,
		ActorWriteLimit:  1,
		TenantAgentLimit: 4,
		AgentLimit:       2,
	}
}

func newMemoryOnlyLimiter(t *testing.T) *Limiter {
	t.Helper()
	limiter, err := NewTenantLimiter(nil, testPolicy())
	if err != nil {
		t.Fatalf("NewTenantLimiter: %v", err)
	}
	limiter.now = func() time.Time { return time.Unix(1_800_000_000, 0).UTC() }
	return limiter
}

func TestTenantNoisyNeighborIsolation(t *testing.T) {
	limiter := newMemoryOnlyLimiter(t)
	ctx := context.Background()
	orgA := uuid.New()
	orgB := uuid.New()
	userA1 := uuid.New()
	userA2 := uuid.New()
	userB := uuid.New()

	for _, userID := range []uuid.UUID{userA1, userA1, userA2, userA2} {
		decision, err := limiter.AllowHuman(ctx, orgA, userID, ClassRead)
		if err != nil || !decision.Allowed {
			t.Fatalf("tenant A request unexpectedly blocked: decision=%+v err=%v", decision, err)
		}
	}
	blocked, err := limiter.AllowHuman(ctx, orgA, uuid.New(), ClassRead)
	if err != nil {
		t.Fatalf("tenant A blocked request: %v", err)
	}
	if blocked.Allowed || blocked.Scope != "tenant" {
		t.Fatalf("tenant A should be aggregate-limited, got %+v", blocked)
	}

	allowed, err := limiter.AllowHuman(ctx, orgB, userB, ClassRead)
	if err != nil {
		t.Fatalf("tenant B request: %v", err)
	}
	if !allowed.Allowed {
		t.Fatalf("tenant B was affected by tenant A budget: %+v", allowed)
	}
}

func TestActorLimitDoesNotBlockSiblingActor(t *testing.T) {
	limiter := newMemoryOnlyLimiter(t)
	ctx := context.Background()
	orgID := uuid.New()
	noisy := uuid.New()
	sibling := uuid.New()

	for i := 0; i < 2; i++ {
		decision, err := limiter.AllowHuman(ctx, orgID, noisy, ClassRead)
		if err != nil || !decision.Allowed {
			t.Fatalf("request %d unexpectedly blocked: %+v %v", i, decision, err)
		}
	}
	blocked, err := limiter.AllowHuman(ctx, orgID, noisy, ClassRead)
	if err != nil || blocked.Allowed || blocked.Scope != "actor" {
		t.Fatalf("noisy actor should be actor-limited: %+v %v", blocked, err)
	}

	allowed, err := limiter.AllowHuman(ctx, orgID, sibling, ClassRead)
	if err != nil || !allowed.Allowed {
		t.Fatalf("sibling actor should retain its own budget: %+v %v", allowed, err)
	}
}

func TestHumanAndAgentBudgetsAreIndependent(t *testing.T) {
	limiter := newMemoryOnlyLimiter(t)
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	for i := 0; i < 2; i++ {
		decision, err := limiter.AllowAgent(ctx, orgID, "sdr-01")
		if err != nil || !decision.Allowed {
			t.Fatalf("agent request %d unexpectedly blocked: %+v %v", i, decision, err)
		}
	}
	blocked, _ := limiter.AllowAgent(ctx, orgID, "sdr-01")
	if blocked.Allowed {
		t.Fatal("agent should have exhausted only its machine budget")
	}

	human, err := limiter.AllowHuman(ctx, orgID, userID, ClassRead)
	if err != nil || !human.Allowed {
		t.Fatalf("agent traffic starved the human namespace: %+v %v", human, err)
	}
}

func TestReadAndWriteBudgetsAreIndependent(t *testing.T) {
	limiter := newMemoryOnlyLimiter(t)
	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	write, err := limiter.AllowHuman(ctx, orgID, userID, ClassWrite)
	if err != nil || !write.Allowed {
		t.Fatalf("first write: %+v %v", write, err)
	}
	blocked, _ := limiter.AllowHuman(ctx, orgID, userID, ClassWrite)
	if blocked.Allowed {
		t.Fatal("second write should hit actor write budget")
	}
	read, err := limiter.AllowHuman(ctx, orgID, userID, ClassRead)
	if err != nil || !read.Allowed {
		t.Fatalf("write budget should not consume read budget: %+v %v", read, err)
	}
}

func TestTenantBucketKeyDoesNotExposeRawIdentity(t *testing.T) {
	org := uuid.New().String()
	key := tenantBucketKey("tenant", ClassRead, org, time.Unix(1_800_000_000, 0).UTC())
	if len(key) == 0 {
		t.Fatal("empty key")
	}
	if contains(key, org) {
		t.Fatalf("redis key leaks raw organization identifier: %s", key)
	}
}

func contains(value, part string) bool {
	if len(part) == 0 || len(value) < len(part) {
		return false
	}
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}

func TestRedisTenantLimitIsSharedAcrossBackendInstances(t *testing.T) {
	client := testClient(t)
	policy := testPolicy()
	first, err := NewTenantLimiter(client, policy)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewTenantLimiter(client, policy)
	if err != nil {
		t.Fatal(err)
	}
	orgA, orgB := uuid.New(), uuid.New()
	userA, userB := uuid.New(), uuid.New()
	ctx := context.Background()

	for i, limiter := range []*Limiter{first, second} {
		decision, err := limiter.AllowHuman(ctx, orgA, userA, ClassRead)
		if err != nil || !decision.Allowed {
			t.Fatalf("instance %d request unexpectedly blocked: %+v %v", i, decision, err)
		}
	}
	blocked, err := first.AllowHuman(ctx, orgA, userA, ClassRead)
	if err != nil || blocked.Allowed {
		t.Fatalf("shared Redis actor budget was not enforced: %+v %v", blocked, err)
	}
	allowedOtherTenant, err := second.AllowHuman(ctx, orgB, userB, ClassRead)
	if err != nil || !allowedOtherTenant.Allowed {
		t.Fatalf("tenant B should remain isolated in Redis: %+v %v", allowedOtherTenant, err)
	}
}
