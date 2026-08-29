// Package ratelimit implements httprate.LimitCounter backed by Redis, so
// request limits survive a backend restart and are shared correctly across
// multiple backend replicas — an in-memory counter (httprate's default)
// resets on every restart and is per-process, so a load-balanced
// deployment effectively multiplies the configured limit by replica count.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-chi/httprate"
	"github.com/redis/go-redis/v9"
)

type RedisCounter struct {
	client       *redis.Client
	windowLength time.Duration
}

// NewRedisCounter returns an httprate.LimitCounter. Call httprate.NewRateLimiter
// with httprate.WithLimitCounter(counter) to use it.
func NewRedisCounter(client *redis.Client) *RedisCounter {
	return &RedisCounter{client: client}
}

func (c *RedisCounter) Config(requestLimit int, windowLength time.Duration) {
	c.windowLength = windowLength
}

func (c *RedisCounter) Increment(key string, currentWindow time.Time) error {
	return c.IncrementBy(key, currentWindow, 1)
}

func (c *RedisCounter) IncrementBy(key string, currentWindow time.Time, amount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	redisKey := bucketKey(key, currentWindow)
	pipe := c.client.TxPipeline()
	incr := pipe.IncrBy(ctx, redisKey, int64(amount))
	// Keep the bucket around for one extra window so Get can still read the
	// previous window's count for httprate's two-bucket rate calculation.
	pipe.Expire(ctx, redisKey, c.windowLength*2)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("ratelimit: increment %s: %w", redisKey, err)
	}
	_ = incr
	return nil
}

func (c *RedisCounter) Get(key string, currentWindow, previousWindow time.Time) (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	currKey := bucketKey(key, currentWindow)
	prevKey := bucketKey(key, previousWindow)

	results, err := c.client.MGet(ctx, prevKey, currKey).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("ratelimit: get %s: %w", key, err)
	}

	// MGet was called as (prevKey, currKey), so results[0] is the previous
	// window's count and results[1] is the current window's — but this
	// method must return (current, previous), matching httprate.LimitCounter.
	return toInt(results[1]), toInt(results[0]), nil
}

func bucketKey(key string, window time.Time) string {
	return fmt.Sprintf("lyrahcle:ratelimit:%d", httprate.LimitCounterKey(key, window))
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case string:
		var n int
		fmt.Sscanf(val, "%d", &n)
		return n
	default:
		return 0
	}
}
