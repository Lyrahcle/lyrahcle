// Package respcache caches JSON-serializable API responses in Redis for a
// short TTL, to avoid recomputing or re-querying data that's identical for
// every caller within that window (SGP4 orbital positions, the space-CVE
// feed that only changes once a day). Every method is nil-receiver-safe:
// a nil *Cache (no Redis configured) behaves as "always a cache miss,
// writes are a no-op" rather than an error.
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

// Get unmarshals a cached value into dest. Returns (false, nil) on a miss
// (including "no Redis configured"), (false, nil) on any Redis error
// (treated as a miss — caching must never turn a transient Redis problem
// into a request failure), and (true, nil) on a hit.
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

// Set stores value under key for ttl. Errors are swallowed (same
// reasoning as Get) — a failed cache write should never fail the request
// that computed the value.
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
