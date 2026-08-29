package tokenblocklist

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Blocklist struct {
	client *redis.Client
}

func New(client *redis.Client) *Blocklist {
	return &Blocklist{client: client}
}

func key(jti string) string {
	return "lyrahcle:revoked-jti:" + jti
}

func (b *Blocklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if b == nil || b.client == nil || jti == "" || ttl <= 0 {
		return nil
	}
	if err := b.client.Set(ctx, key(jti), "1", ttl).Err(); err != nil {
		return fmt.Errorf("tokenblocklist: revoke: %w", err)
	}
	return nil
}

func (b *Blocklist) IsRevoked(ctx context.Context, jti string) bool {
	if b == nil || b.client == nil || jti == "" {
		return false
	}
	n, err := b.client.Exists(ctx, key(jti)).Result()
	if err != nil {
		return false
	}
	return n > 0
}
