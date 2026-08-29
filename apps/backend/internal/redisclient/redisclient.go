package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

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
