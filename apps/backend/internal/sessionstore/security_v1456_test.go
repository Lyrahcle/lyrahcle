package sessionstore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func unavailableClient(t *testing.T) *redis.Client {
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

func TestRegisterRejectsInvalidSessionsWithoutRedis(t *testing.T) {
	store := New(unavailableClient(t))
	ctx := context.Background()
	now := time.Now()

	tests := []Session{
		{UserID: "user", ExpiresAt: now.Add(time.Hour)},
		{ID: "session", ExpiresAt: now.Add(time.Hour)},
		{ID: "session", UserID: "user"},
		{ID: "session", UserID: "user", ExpiresAt: now.Add(-time.Second)},
	}
	for i, session := range tests {
		if err := store.Register(ctx, session); err == nil || !strings.Contains(err.Error(), "invalid session") {
			t.Fatalf("case %d error=%v want invalid session", i, err)
		}
	}
}

func TestSessionKeyNamespacesDoNotOverlap(t *testing.T) {
	if sessionKey("abc") == userSessionsKey("abc") {
		t.Fatal("session and user-session namespaces must not overlap")
	}
	if !strings.HasPrefix(sessionKey("abc"), "lyrahcle:session:") {
		t.Fatal("unexpected session key namespace")
	}
	if !strings.HasPrefix(userSessionsKey("abc"), "lyrahcle:user-sessions:") {
		t.Fatal("unexpected user-session key namespace")
	}
}

func TestNormalizeUserAgentTrimsAndCapsLength(t *testing.T) {
	got := normalizeUserAgent("  browser  ")
	if got != "browser" {
		t.Fatalf("got %q want browser", got)
	}
	long := strings.Repeat("x", 200)
	if len(normalizeUserAgent(long)) != 160 {
		t.Fatal("user agent must be capped at 160 bytes")
	}
}

func TestUnavailableStoreMethodsFailClosed(t *testing.T) {
	var nilStore *Store
	ctx := context.Background()

	if _, err := nilStore.Get(ctx, "session"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil Get error=%v", err)
	}
	if err := nilStore.Delete(ctx, "user", "session"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil Delete error=%v", err)
	}
	if err := nilStore.DeleteUser(ctx, "user"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil DeleteUser error=%v", err)
	}

	store := New(nil)
	if _, err := store.Get(ctx, "session"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil-client Get error=%v", err)
	}
	if err := store.Delete(ctx, "user", "session"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil-client Delete error=%v", err)
	}
}

func TestRedisErrorsArePropagated(t *testing.T) {
	store := New(unavailableClient(t))
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	valid := Session{ID: "s", UserID: "u", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	if err := store.Register(ctx, valid); err == nil {
		t.Fatal("Register must surface Redis outage")
	}
	if _, err := store.Get(ctx, "s"); err == nil {
		t.Fatal("Get must surface Redis outage")
	}
	if _, err := store.List(ctx, "u"); err == nil {
		t.Fatal("List must surface Redis outage")
	}
	if err := store.Delete(ctx, "u", "s"); err == nil {
		t.Fatal("Delete must surface Redis outage")
	}
	if err := store.DeleteUser(ctx, "u"); err == nil {
		t.Fatal("DeleteUser must surface Redis outage")
	}
}

func TestDeleteRejectsCrossUserSession(t *testing.T) {
	client := testClient(t)
	store := New(client)
	ctx := context.Background()
	owner := "owner-" + t.Name()
	attacker := "other-" + t.Name()
	id := "session-owner-" + t.Name()
	session := Session{ID: id, UserID: owner, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Minute).UTC()}
	if err := store.Register(ctx, session); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(ctx, attacker, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user Delete error=%v want ErrNotFound", err)
	}
	got, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("session should remain after cross-user delete attempt: %v", err)
	}
	if got.UserID != owner {
		t.Fatalf("session owner=%q want %q", got.UserID, owner)
	}
}

func TestGetRejectsCorruptSessionPayload(t *testing.T) {
	client := testClient(t)
	ctx := context.Background()
	id := "corrupt-" + t.Name()
	if err := client.Set(ctx, sessionKey(id), "{", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := New(client).Get(ctx, id); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("error=%v want decode failure", err)
	}
}

func TestListSkipsMissingAndForeignSessions(t *testing.T) {
	client := testClient(t)
	store := New(client)
	ctx := context.Background()
	userID := "list-user-" + t.Name()
	now := time.Now().UTC()

	foreign := Session{ID: "foreign-" + t.Name(), UserID: "different-user", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}
	payload, err := json.Marshal(foreign)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, sessionKey(foreign.ID), payload, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if err := client.ZAdd(ctx, userSessionsKey(userID),
		redis.Z{Score: float64(now.Add(time.Minute).Unix()), Member: foreign.ID},
		redis.Z{Score: float64(now.Add(time.Minute).Unix()), Member: "missing-" + t.Name()},
	).Err(); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list=%+v want empty", list)
	}
}
