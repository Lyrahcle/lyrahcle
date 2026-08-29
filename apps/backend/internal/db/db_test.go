package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS schema_migrations CASCADE`); err != nil {
		t.Fatalf("failed to reset schema_migrations: %v", err)
	}

	return pool
}

func TestMigrateAppliesAndRecordsEachFile(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	if err := Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("failed to count schema_migrations: %v", err)
	}
	if count != len(entries) {
		t.Errorf("schema_migrations has %d rows, want %d (one per migration file)", count, len(entries))
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	if err := Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("first Migrate returned error: %v", err)
	}
	if err := Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("second Migrate (should be a no-op) returned error: %v", err)
	}
}

func TestMigrateSkipsAlreadyAppliedFiles(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	if err := Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}

	var firstAppliedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT applied_at FROM schema_migrations ORDER BY filename LIMIT 1`).Scan(&firstAppliedAt); err != nil {
		t.Fatalf("failed to read applied_at: %v", err)
	}

	if err := Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("second Migrate returned error: %v", err)
	}

	var secondAppliedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT applied_at FROM schema_migrations ORDER BY filename LIMIT 1`).Scan(&secondAppliedAt); err != nil {
		t.Fatalf("failed to read applied_at after re-run: %v", err)
	}

	if !firstAppliedAt.Equal(secondAppliedAt) {
		t.Errorf("applied_at changed after re-running Migrate (%v -> %v); the migration was re-executed instead of skipped", firstAppliedAt, secondAppliedAt)
	}
}
