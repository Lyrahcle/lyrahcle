// Package tokenblocklist lets the backend revoke a specific JWT before its
// natural expiry — on logout, and when a password changes — despite JWTs
// otherwise being stateless and valid until they expire on their own.
//
// A blocklist entry's TTL always matches the token's remaining lifetime:
// once the token would have expired anyway, the entry is redundant and
// Redis evicts it for free.
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

// New wraps client for revocation checks. client may be nil — every method
// on Blocklist is nil-receiver-safe and treats that as "no Redis
// configured", so revocation is simply not enforced rather than erroring.
func New(client *redis.Client) *Blocklist {
	return &Blocklist{client: client}
}

func key(jti string) string {
	return "lyrahcle:revoked-jti:" + jti
}

// Revoke blocks jti for ttl (normally the token's remaining time to
// expiry). A no-op if jti is empty (an older token minted before jti
// support) or the blocklist has no Redis client.
func (b *Blocklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if b == nil || b.client == nil || jti == "" || ttl <= 0 {
		return nil
	}
	if err := b.client.Set(ctx, key(jti), "1", ttl).Err(); err != nil {
		return fmt.Errorf("tokenblocklist: revoke: %w", err)
	}
	return nil
}

// IsRevoked reports whether jti has been revoked. Always false if the
// blocklist has no Redis client (revocation not enforced) or on a Redis
// error (fails open — a transient Redis hiccup must not lock everyone
// out; the JWT signature/expiry check is still enforced regardless).
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
