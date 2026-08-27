package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pgx connection pool using the given DSN.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// Migrate applies migrations/*.sql in lexical order.
// Minimal implementation for Phase 3; swap for golang-migrate once
// schema churn increases and rollbacks are needed.
func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("db: read migrations dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := dir + "/" + e.Name()
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("db: read %s: %w", path, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("db: exec %s: %w", path, err)
		}
	}
	return nil
}
