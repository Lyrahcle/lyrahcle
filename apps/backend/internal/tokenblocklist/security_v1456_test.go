package tokenblocklist

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func unavailableClientV1456(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  25 * time.Millisecond,
		ReadTimeout:  25 * time.Millisecond,
		WriteTimeout: 25 * time.Millisecond,
		MaxRetries:   0,
	})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestKeyNamespacesAreSeparated(t *testing.T) {
	if key("x") == userKey("x") {
		t.Fatal("per-token and per-user revocation namespaces must differ")
	}
	if !strings.HasPrefix(key("x"), "lyrahcle:revoked-jti:") {
		t.Fatal("unexpected JTI namespace")
	}
	if !strings.HasPrefix(userKey("x"), "lyrahcle:revoked-user-before:") {
		t.Fatal("unexpected user cutoff namespace")
	}
}

func TestRevokeNoopsForInvalidInputs(t *testing.T) {
	client := unavailableClientV1456(t)
	b := New(client)
	ctx := context.Background()

	for _, tc := range []struct {
		jti string
		ttl time.Duration
	}{{"", time.Minute}, {"jti", 0}, {"jti", -time.Second}} {
		if err := b.Revoke(ctx, tc.jti, tc.ttl); err != nil {
			t.Fatalf("Revoke(%q,%s)=%v want nil", tc.jti, tc.ttl, err)
		}
	}
	for _, tc := range []struct {
		user string
		ttl  time.Duration
	}{{"", time.Minute}, {"user", 0}, {"user", -time.Second}} {
		if err := b.RevokeUser(ctx, tc.user, time.Now(), tc.ttl); err != nil {
			t.Fatalf("RevokeUser(%q,%s)=%v want nil", tc.user, tc.ttl, err)
		}
	}
}

func TestRedisOutageIsReturnedBySecurityCriticalChecks(t *testing.T) {
	b := New(unavailableClientV1456(t))
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	if err := b.Revoke(ctx, "jti", time.Minute); err == nil {
		t.Fatal("Revoke must report Redis outage")
	}
	if err := b.RevokeUser(ctx, "user", time.Now(), time.Minute); err == nil {
		t.Fatal("RevokeUser must report Redis outage")
	}
	if _, err := b.Check(ctx, "jti", "", time.Time{}); err == nil {
		t.Fatal("JTI Check must report Redis outage")
	}
	if _, err := b.Check(ctx, "", "user", time.Now()); err == nil {
		t.Fatal("user cutoff Check must report Redis outage")
	}
}

func TestCheckRejectsCorruptUserCutoff(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	userID := "corrupt-cutoff-" + uuid.NewString()
	if err := client.Set(ctx, userKey(userID), "not-a-number", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	revoked, err := New(client).Check(ctx, "", userID, time.Now())
	if err == nil || revoked {
		t.Fatalf("Check=(%v,%v) want false,error", revoked, err)
	}
	if !strings.Contains(err.Error(), "invalid user cutoff") {
		t.Fatalf("error=%v want invalid user cutoff", err)
	}
}

func TestPerTokenRevocationTakesPrecedenceOverUserState(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	b := New(client)
	jti := "precedence-" + uuid.NewString()
	if err := b.Revoke(ctx, jti, time.Minute); err != nil {
		t.Fatal(err)
	}

	revoked, err := b.Check(ctx, jti, "user", time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("explicit JTI revocation must win even for a newer token timestamp")
	}
}

func TestUserRevocationBoundaryIsInclusive(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	b := New(client)
	userID := "boundary-" + uuid.NewString()
	cutoff := time.Now().UTC().Truncate(time.Microsecond)
	if err := b.RevokeUser(ctx, userID, cutoff, time.Minute); err != nil {
		t.Fatal(err)
	}

	revoked, err := b.Check(ctx, "", userID, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("token issued exactly at cutoff must be revoked")
	}
	revoked, err = b.Check(ctx, "", userID, cutoff.Add(time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	if revoked {
		t.Fatal("token issued after cutoff must remain valid")
	}
}
