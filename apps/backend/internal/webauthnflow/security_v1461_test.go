package webauthnflow

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func webAuthnTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set, skipping Redis integration test")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("connect to test redis at %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestStoreFailsClosedWithoutRedis(t *testing.T) {
	store := New(nil)
	if store.ttl != 5*time.Minute {
		t.Fatalf("ttl = %s, want 5m", store.ttl)
	}
	if _, err := store.Create(context.Background(), Ceremony{}); err == nil {
		t.Fatal("Create without Redis succeeded")
	}
	if _, err := store.Consume(context.Background(), uuid.NewString(), "login"); err == nil {
		t.Fatal("Consume without Redis succeeded")
	}
}

func TestConsumeRejectsMalformedIDBeforeRedisAccess(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer client.Close()
	if _, err := New(client).Consume(context.Background(), "not-a-uuid", "login"); err == nil || err.Error() != "invalid ceremony id" {
		t.Fatalf("Consume error = %v, want invalid ceremony id", err)
	}
}

func TestUserImplementsWebAuthnIdentity(t *testing.T) {
	credential := webauthn.Credential{ID: []byte("credential")}
	user := User{
		Handle:      []byte("user-handle"),
		Name:        "operator@example.test",
		DisplayName: "Operator",
		Credentials: []webauthn.Credential{credential},
	}
	if got := string(user.WebAuthnID()); got != "user-handle" {
		t.Fatalf("WebAuthnID = %q", got)
	}
	if got := user.WebAuthnName(); got != user.Name {
		t.Fatalf("WebAuthnName = %q", got)
	}
	if got := user.WebAuthnDisplayName(); got != user.DisplayName {
		t.Fatalf("WebAuthnDisplayName = %q", got)
	}
	if got := user.WebAuthnCredentials(); len(got) != 1 || string(got[0].ID) != "credential" {
		t.Fatalf("WebAuthnCredentials = %#v", got)
	}
}

func TestCreateConsumeIsOneTime(t *testing.T) {
	client := webAuthnTestRedis(t)
	store := New(client)
	ctx := context.Background()
	want := Ceremony{UserID: uuid.New(), Purpose: "login", Label: "security-key"}
	id, err := store.Create(ctx, want)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("Create id = %q: %v", id, err)
	}
	got, err := store.Consume(ctx, id, "login")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if got.UserID != want.UserID || got.Purpose != want.Purpose || got.Label != want.Label {
		t.Fatalf("Consume = %#v, want %#v", got, want)
	}
	if _, err := store.Consume(ctx, id, "login"); err == nil {
		t.Fatal("second Consume succeeded; ceremony must be one-time")
	}
}

func TestPurposeMismatchDoesNotDestroyCeremony(t *testing.T) {
	client := webAuthnTestRedis(t)
	store := New(client)
	ctx := context.Background()
	id, err := store.Create(ctx, Ceremony{UserID: uuid.New(), Purpose: "login"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Consume(ctx, id, "register"); err == nil || err.Error() != "webauthn ceremony mismatch" {
		t.Fatalf("mismatched Consume error = %v", err)
	}
	if _, err := store.Consume(ctx, id, "login"); err != nil {
		t.Fatalf("legitimate Consume after mismatch: %v", err)
	}
}

func TestInvalidCeremonyPayloadFailsClosedWithoutConsumption(t *testing.T) {
	client := webAuthnTestRedis(t)
	store := New(client)
	ctx := context.Background()

	corruptID := uuid.NewString()
	if err := client.Set(ctx, ceremonyKey(corruptID), "{", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Consume(ctx, corruptID, "login"); err == nil {
		t.Fatal("corrupt ceremony payload was accepted")
	}
	if exists, err := client.Exists(ctx, ceremonyKey(corruptID)).Result(); err != nil || exists != 1 {
		t.Fatalf("corrupt ceremony should remain TTL-bound for diagnosis: exists=%d err=%v", exists, err)
	}

	nilUserID := uuid.NewString()
	raw := `{"user_id":"00000000-0000-0000-0000-000000000000","purpose":"login","session":{}}`
	if err := client.Set(ctx, ceremonyKey(nilUserID), raw, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Consume(ctx, nilUserID, "login"); err == nil || err.Error() != "webauthn ceremony mismatch" {
		t.Fatalf("nil-user ceremony error = %v", err)
	}
}

func TestClosedRedisClientFailsClosed(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	store := New(client)
	if _, err := store.Create(context.Background(), Ceremony{UserID: uuid.New(), Purpose: "login"}); err == nil {
		t.Fatal("Create with closed Redis client succeeded")
	}
	if _, err := store.Consume(context.Background(), uuid.NewString(), "login"); err == nil {
		t.Fatal("Consume with closed Redis client succeeded")
	}
}

func TestConsumeLosesRaceSafely(t *testing.T) {
	client := webAuthnTestRedis(t)
	store := New(client)
	ctx := context.Background()
	id, err := store.Create(ctx, Ceremony{UserID: uuid.New(), Purpose: "login"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := client.Get(ctx, ceremonyKey(id)).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, ceremonyKey(id), append(raw, ' '), time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	deleted, err := consumeCeremonyScript.Run(ctx, client, []string{ceremonyKey(id)}, raw).Int64()
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatalf("stale ceremony payload deleted replacement: %d", deleted)
	}
}

func TestCeremonyKeyIsNamespaced(t *testing.T) {
	id := uuid.NewString()
	if got, want := ceremonyKey(id), "lyrahcle:webauthn:"+id; got != want {
		t.Fatalf("ceremonyKey = %q, want %q", got, want)
	}
}
