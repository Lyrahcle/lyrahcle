package respcache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func New(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return false, nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, nil
	}
	return true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	if c == nil || c.client == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	c.client.Set(ctx, key, raw, ttl)
}
func (c *Cache) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, key).Err()
}
