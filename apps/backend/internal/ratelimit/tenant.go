package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
)

const (
	ClassRead  = "read"
	ClassWrite = "write"
	ClassAgent = "agent"
)

// Policy keeps authenticated rate budgets independent from the coarse platform
// ceiling. Tenant budgets protect noisy-neighbor isolation; actor budgets stop a
// single user/agent from consuming the whole tenant allowance.
type Policy struct {
	Window           time.Duration
	TenantReadLimit  int
	ActorReadLimit   int
	TenantWriteLimit int
	ActorWriteLimit  int
	TenantAgentLimit int
	AgentLimit       int
}

func (p Policy) Validate() error {
	if p.Window < time.Second || p.Window > time.Hour {
		return errors.New("ratelimit: window must be between 1s and 1h")
	}
	values := map[string]int{
		"tenant_read":  p.TenantReadLimit,
		"actor_read":   p.ActorReadLimit,
		"tenant_write": p.TenantWriteLimit,
		"actor_write":  p.ActorWriteLimit,
		"tenant_agent": p.TenantAgentLimit,
		"agent":        p.AgentLimit,
	}
	for name, value := range values {
		if value < 1 {
			return fmt.Errorf("ratelimit: %s limit must be positive", name)
		}
	}
	if p.ActorReadLimit > p.TenantReadLimit || p.ActorWriteLimit > p.TenantWriteLimit || p.AgentLimit > p.TenantAgentLimit {
		return errors.New("ratelimit: actor limits must not exceed tenant limits")
	}
	return nil
}

type Decision struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
	Scope      string
	Class      string
}

type memoryBucket struct {
	count     int
	expiresAt time.Time
}

type memoryCounter struct {
	mu      sync.Mutex
	buckets map[string]memoryBucket
	ops     uint64
}

func newMemoryCounter() *memoryCounter {
	return &memoryCounter{buckets: make(map[string]memoryBucket)}
}

func (m *memoryCounter) increment(key string, now time.Time, window time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ops++
	if m.ops%256 == 0 {
		for existing, bucket := range m.buckets {
			if !bucket.expiresAt.After(now) {
				delete(m.buckets, existing)
			}
		}
	}

	bucket := m.buckets[key]
	if !bucket.expiresAt.After(now) {
		bucket = memoryBucket{expiresAt: now.Truncate(window).Add(window)}
	}
	bucket.count++
	m.buckets[key] = bucket
	return bucket.count
}

type Limiter struct {
	redis  *redis.Client
	memory *memoryCounter
	policy Policy
	now    func() time.Time
}

func NewTenantLimiter(client *redis.Client, policy Policy) (*Limiter, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Limiter{redis: client, memory: newMemoryCounter(), policy: policy, now: time.Now}, nil
}

func (l *Limiter) AllowHuman(ctx context.Context, organizationID, userID uuid.UUID, class string) (Decision, error) {
	if organizationID == uuid.Nil || userID == uuid.Nil {
		return Decision{}, errors.New("ratelimit: organization and user are required")
	}
	var tenantLimit, actorLimit int
	switch class {
	case ClassRead:
		tenantLimit, actorLimit = l.policy.TenantReadLimit, l.policy.ActorReadLimit
	case ClassWrite:
		tenantLimit, actorLimit = l.policy.TenantWriteLimit, l.policy.ActorWriteLimit
	default:
		return Decision{}, fmt.Errorf("ratelimit: unsupported human class %q", class)
	}

	now := l.now().UTC()
	tenantDecision, err := l.allowAt(ctx, "tenant", class, organizationID.String(), tenantLimit, now)
	if err != nil {
		return Decision{}, err
	}
	if !tenantDecision.Allowed {
		metrics.RateLimitDecisions.WithLabelValues("tenant", class, "blocked").Inc()
		return tenantDecision, nil
	}
	actorSubject := organizationID.String() + ":" + userID.String()
	actorDecision, err := l.allowAt(ctx, "actor", class, actorSubject, actorLimit, now)
	if err != nil {
		return Decision{}, err
	}
	if !actorDecision.Allowed {
		metrics.RateLimitDecisions.WithLabelValues("actor", class, "blocked").Inc()
		return actorDecision, nil
	}
	metrics.RateLimitDecisions.WithLabelValues("actor", class, "allowed").Inc()
	return actorDecision, nil
}

func (l *Limiter) AllowAgent(ctx context.Context, organizationID uuid.UUID, agentID string) (Decision, error) {
	if organizationID == uuid.Nil || agentID == "" {
		return Decision{}, errors.New("ratelimit: organization and agent are required")
	}
	now := l.now().UTC()
	tenantDecision, err := l.allowAt(ctx, "tenant_agent", ClassAgent, organizationID.String(), l.policy.TenantAgentLimit, now)
	if err != nil {
		return Decision{}, err
	}
	if !tenantDecision.Allowed {
		metrics.RateLimitDecisions.WithLabelValues("tenant_agent", ClassAgent, "blocked").Inc()
		return tenantDecision, nil
	}
	agentDecision, err := l.allowAt(ctx, "agent", ClassAgent, organizationID.String()+":"+agentID, l.policy.AgentLimit, now)
	if err != nil {
		return Decision{}, err
	}
	result := "allowed"
	if !agentDecision.Allowed {
		result = "blocked"
	}
	metrics.RateLimitDecisions.WithLabelValues("agent", ClassAgent, result).Inc()
	return agentDecision, nil
}

func (l *Limiter) allow(ctx context.Context, scope, class, subject string, limit int) (Decision, error) {
	return l.allowAt(ctx, scope, class, subject, limit, l.now().UTC())
}

func (l *Limiter) allowAt(ctx context.Context, scope, class, subject string, limit int, now time.Time) (Decision, error) {
	windowStart := now.Truncate(l.policy.Window)
	windowEnd := windowStart.Add(l.policy.Window)
	key := tenantBucketKey(scope, class, subject, windowStart)

	count, redisErr := l.incrementRedis(ctx, key)
	if redisErr != nil {
		metrics.RateLimitBackendFallback.WithLabelValues("redis", scope).Inc()
		count = l.memory.increment(key, now, l.policy.Window)
	}
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	return Decision{
		Allowed:    count <= limit,
		Limit:      limit,
		Remaining:  remaining,
		RetryAfter: windowEnd.Sub(now).Round(time.Second),
		Scope:      scope,
		Class:      class,
	}, nil
}

var incrementScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)

func (l *Limiter) incrementRedis(ctx context.Context, key string) (int, error) {
	if l.redis == nil {
		return 0, errors.New("redis unavailable")
	}
	result, err := incrementScript.Run(ctx, l.redis, []string{key}, l.policy.Window.Milliseconds()).Int()
	if err != nil {
		return 0, err
	}
	return result, nil
}

func tenantBucketKey(scope, class, subject string, window time.Time) string {
	digest := sha256.Sum256([]byte(subject))
	return fmt.Sprintf("lyrahcle:tenant-rate:%s:%s:%d:%s", scope, class, window.Unix(), hex.EncodeToString(digest[:16]))
}
