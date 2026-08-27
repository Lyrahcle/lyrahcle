package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	schema := `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;

		DROP TABLE IF EXISTS users CASCADE;
		DROP TABLE IF EXISTS satellites CASCADE;
		DROP TABLE IF EXISTS assets CASCADE;
		DROP TABLE IF EXISTS programs CASCADE;

		CREATE TABLE programs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL
		);

		CREATE TABLE assets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			program_id UUID REFERENCES programs(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			value TEXT NOT NULL,
			first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			UNIQUE (program_id, kind, value)
		);

		CREATE TABLE satellites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			norad_id INTEGER NOT NULL UNIQUE,
			name TEXT NOT NULL,
			tle_line1 TEXT NOT NULL,
			tle_line2 TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'viewer',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`
	if _, err := pool.Exec(ctx, schema); err != nil {
		t.Fatalf("failed to set up test schema: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestAssetStoreUpsertAndList(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAssetStore(pool)

	var programID uuid.UUID
	err := pool.QueryRow(ctx, `INSERT INTO programs (name) VALUES ('test-program') RETURNING id`).Scan(&programID)
	if err != nil {
		t.Fatalf("failed to insert test program: %v", err)
	}

	if err := store.Upsert(ctx, programID, "domain", "example.com"); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if err := store.Upsert(ctx, programID, "domain", "example.com"); err != nil {
		t.Fatalf("second Upsert (should update last_seen) returned error: %v", err)
	}

	assets, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("got %d assets, want 1 (upsert should not duplicate)", len(assets))
	}
	if assets[0].Value != "example.com" {
		t.Errorf("asset value = %q, want %q", assets[0].Value, "example.com")
	}
}

func TestSatelliteStoreUpsertAndList(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewSatelliteStore(pool)

	line1 := "1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995"
	line2 := "2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754"

	if err := store.Upsert(ctx, 25544, "ISS (ZARYA)", line1, line2); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	updatedLine1 := "1 25544U 98067A   26240.00000000  .00008717  00000+0  16243-3 0  9996"
	if err := store.Upsert(ctx, 25544, "ISS (ZARYA)", updatedLine1, line2); err != nil {
		t.Fatalf("second Upsert (should update TLE) returned error: %v", err)
	}

	satellites, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(satellites) != 1 {
		t.Fatalf("got %d satellites, want 1 (upsert on norad_id should not duplicate)", len(satellites))
	}
	if satellites[0].TLELine1 != updatedLine1 {
		t.Errorf("TLE line1 was not updated by the second upsert")
	}
}

func TestUserStoreCreateAndGetByEmail(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewUserStore(pool)

	user, err := store.Create(ctx, "user@example.com", "hashed-password", "viewer")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "user@example.com")
	}

	fetched, err := store.GetByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("GetByEmail returned error: %v", err)
	}
	if fetched.ID != user.ID {
		t.Errorf("GetByEmail returned a different user ID than Create")
	}

	if _, err := store.Create(ctx, "user@example.com", "another-hash", "viewer"); err != ErrEmailTaken {
		t.Errorf("Create with duplicate email: err = %v, want ErrEmailTaken", err)
	}

	if _, err := store.GetByEmail(ctx, "nobody@example.com"); err != ErrUserNotFound {
		t.Errorf("GetByEmail for missing user: err = %v, want ErrUserNotFound", err)
	}
}

func TestUserStoreCount(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewUserStore(pool)

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("Count on empty table = %d, want 0", count)
	}

	if _, err := store.Create(ctx, "a@example.com", "hash", "admin"); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("Count after one insert = %d, want 1", count)
	}
}
