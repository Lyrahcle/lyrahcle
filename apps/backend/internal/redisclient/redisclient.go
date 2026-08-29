// Package redisclient connects to Redis for the backend's optional
// features (Redis-backed rate limiting, JWT revocation, response caching).
// None of these are required for the backend to run: Connect is called
// best-effort at startup, and every caller in this codebase treats a nil
// *redis.Client as "feature disabled" rather than an error, so a
// self-hosted deployment without Redis keeps working exactly as before.
package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Connect pings addr and returns a ready client, or an error if Redis
// isn't reachable within timeout. Callers should treat a connect failure
// as non-fatal and continue with a nil client.
func Connect(addr string, timeout time.Duration) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redisclient: ping %s: %w", addr, err)
	}

	return client, nil
}
