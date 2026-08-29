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
	namespace    string
	windowLength time.Duration
}

func NewRedisCounter(client *redis.Client, namespace string) *RedisCounter {
	return &RedisCounter{client: client, namespace: namespace}
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

	redisKey := c.bucketKey(key, currentWindow)
	pipe := c.client.TxPipeline()
	incr := pipe.IncrBy(ctx, redisKey, int64(amount))
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

	currKey := c.bucketKey(key, currentWindow)
	prevKey := c.bucketKey(key, previousWindow)

	results, err := c.client.MGet(ctx, prevKey, currKey).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("ratelimit: get %s: %w", key, err)
	}

	return toInt(results[1]), toInt(results[0]), nil
}

func (c *RedisCounter) bucketKey(key string, window time.Time) string {
	return fmt.Sprintf("lyrahcle:ratelimit:%s:%d", c.namespace, httprate.LimitCounterKey(key, window))
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
