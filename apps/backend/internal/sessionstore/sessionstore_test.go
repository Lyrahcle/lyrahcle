package sessionstore

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func testClient(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set, skipping Redis integration test")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("connect to test redis at %s: %v", addr, err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestRegisterListAndDeleteSession(t *testing.T) {
	store := New(testClient(t))
	ctx := context.Background()
	userID := "user-" + t.Name()
	session := Session{
		ID:        "session-" + t.Name(),
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().Add(time.Minute).UTC(),
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
	}
	if err := store.Register(ctx, session); err != nil {
		t.Fatalf("Register: %v", err)
	}

	list, err := store.List(ctx, userID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != session.ID {
		t.Fatalf("sessions = %#v", list)
	}

	if err := store.Delete(ctx, userID, session.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, session.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete error = %v, want ErrNotFound", err)
	}
}

func TestDeleteUserRemovesAllSessions(t *testing.T) {
	store := New(testClient(t))
	ctx := context.Background()
	userID := "user-all-" + t.Name()
	for _, id := range []string{"a-" + t.Name(), "b-" + t.Name()} {
		if err := store.Register(ctx, Session{ID: id, UserID: userID, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Minute).UTC()}); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}
	if err := store.DeleteUser(ctx, userID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	list, err := store.List(ctx, userID)
	if err != nil {
		t.Fatalf("List after DeleteUser: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("sessions = %#v, want empty", list)
	}
}

func TestUnavailableStoreFailsClosed(t *testing.T) {
	store := New(nil)
	ctx := context.Background()
	if err := store.Register(ctx, Session{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Register error = %v, want ErrUnavailable", err)
	}
	if _, err := store.List(ctx, "user"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("List error = %v, want ErrUnavailable", err)
	}
}

func TestDeleteUserUnavailableFailsClosed(t *testing.T) {
	store := New(nil)
	if err := store.DeleteUser(context.Background(), "user"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("DeleteUser error = %v, want ErrUnavailable", err)
	}
}
