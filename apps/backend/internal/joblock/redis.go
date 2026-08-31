package joblock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultPrefix = "lyrahcle:job-lock:"

var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

var renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

type Locker struct {
	client *redis.Client
	prefix string
}

func New(client *redis.Client) *Locker {
	return &Locker{client: client, prefix: defaultPrefix}
}

// Try acquires a distributed lease for a scheduled job. The unlock function
// only releases a lease still owned by this process, so an expired lease that
// has already been acquired by another replica cannot be deleted accidentally.
// While held, the lease is renewed periodically so jobs that legitimately run
// longer than their initial TTL do not become eligible on another replica.
// Renewal and release are token-guarded, so a stale owner cannot extend or
// remove a replacement lease. With no Redis client (development only), it runs
// locally; production already requires Redis at startup.
func (l *Locker) Try(ctx context.Context, job string, ttl time.Duration) (unlock func(context.Context) error, acquired bool, err error) {
	if job == "" {
		return nil, false, errors.New("joblock: empty job name")
	}
	if ttl <= 0 {
		return nil, false, errors.New("joblock: ttl must be positive")
	}
	if l == nil || l.client == nil {
		return func(context.Context) error { return nil }, true, nil
	}

	key := l.prefix + job
	token := uuid.NewString()
	ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, fmt.Errorf("joblock: acquire %s: %w", job, err)
	}
	if !ok {
		return nil, false, nil
	}

	stopRenewal := make(chan struct{})
	renewalDone := make(chan struct{})
	go l.renewUntilStopped(key, token, ttl, stopRenewal, renewalDone)

	var stopOnce sync.Once
	return func(releaseCtx context.Context) error {
		stopOnce.Do(func() { close(stopRenewal) })
		<-renewalDone
		if _, err := releaseScript.Run(releaseCtx, l.client, []string{key}, token).Result(); err != nil && err != redis.Nil {
			return fmt.Errorf("joblock: release %s: %w", job, err)
		}
		return nil
	}, true, nil
}

func (l *Locker) renewUntilStopped(key, token string, ttl time.Duration, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	interval := ttl / 3
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	if interval >= ttl {
		interval = ttl / 2
	}
	if interval <= 0 {
		interval = time.Millisecond
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			ttlMillis := ttl.Milliseconds()
			if ttlMillis < 1 {
				ttlMillis = 1
			}
			renewCtx, cancel := context.WithTimeout(context.Background(), interval)
			result, err := renewScript.Run(renewCtx, l.client, []string{key}, token, ttlMillis).Int()
			cancel()
			if err == nil && result == 0 {
				return
			}
			// A transient Redis error does not immediately abandon renewal.
			// If the lease expires and another replica acquires it, the
			// token-guarded script returns 0 on the next successful check.
			timer.Reset(interval)
		}
	}
}
