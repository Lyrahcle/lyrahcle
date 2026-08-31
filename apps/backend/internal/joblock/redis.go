package joblock

import (
	"context"
	"errors"
	"fmt"
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
// With no Redis client (development only), it runs locally; production already
// requires Redis at startup.
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
	return func(releaseCtx context.Context) error {
		if _, err := releaseScript.Run(releaseCtx, l.client, []string{key}, token).Result(); err != nil && err != redis.Nil {
			return fmt.Errorf("joblock: release %s: %w", job, err)
		}
		return nil
	}, true, nil
}
