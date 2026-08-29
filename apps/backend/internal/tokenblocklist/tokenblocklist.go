package tokenblocklist

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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

func userKey(userID string) string {
	return "lyrahcle:revoked-user-before:" + userID
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

func (b *Blocklist) RevokeUser(ctx context.Context, userID string, issuedBefore time.Time, ttl time.Duration) error {
	if b == nil || b.client == nil || userID == "" || ttl <= 0 {
		return nil
	}
	value := strconv.FormatInt(issuedBefore.UTC().UnixNano(), 10)
	if err := b.client.Set(ctx, userKey(userID), value, ttl).Err(); err != nil {
		return fmt.Errorf("tokenblocklist: revoke user: %w", err)
	}
	return nil
}

func (b *Blocklist) IsRevoked(ctx context.Context, jti string) bool {
	revoked, err := b.Check(ctx, jti, "", time.Time{})
	return err == nil && revoked
}

func (b *Blocklist) Check(ctx context.Context, jti, userID string, issuedAt time.Time) (bool, error) {
	if b == nil || b.client == nil {
		return false, nil
	}
	if jti != "" {
		n, err := b.client.Exists(ctx, key(jti)).Result()
		if err != nil {
			return false, fmt.Errorf("tokenblocklist: check jti: %w", err)
		}
		if n > 0 {
			return true, nil
		}
	}
	if userID == "" || issuedAt.IsZero() {
		return false, nil
	}
	value, err := b.client.Get(ctx, userKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("tokenblocklist: check user: %w", err)
	}
	cutoffNanos, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false, fmt.Errorf("tokenblocklist: invalid user cutoff: %w", err)
	}
	cutoff := time.Unix(0, cutoffNanos).UTC()
	return !issuedAt.UTC().After(cutoff), nil
}
