package db

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mlko01/lyrahcle/apps/backend/internal/dbscope"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
)

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}
	cfg.ConnConfig.Tracer = metrics.DBQueryTracer{}
	cfg.BeforeAcquire = func(acquireCtx context.Context, conn *pgx.Conn) bool {
		request := dbscope.IsRequest(acquireCtx) && !dbscope.IsPlatformControlPlane(acquireCtx)
		organizationID, hasOrganization := dbscope.OrganizationID(acquireCtx)
		requestValue := "off"
		organizationValue := ""
		if request {
			requestValue = "on"
			if hasOrganization {
				organizationValue = organizationID.String()
			}
		}
		_, err := conn.Exec(acquireCtx, `SELECT set_config('lyrahcle.rls_request', $1, false), set_config('lyrahcle.organization_id', $2, false)`, requestValue, organizationValue)
		return err == nil
	}
	cfg.AfterRelease = func(conn *pgx.Conn) bool {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		_, err := conn.Exec(cleanupCtx, `RESET lyrahcle.rls_request; RESET lyrahcle.organization_id`)
		return err == nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

const migrationAdvisoryLockID int64 = 0x4c59524148434c45
const cleanupTimeout = 2 * time.Second

func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("db: acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationAdvisoryLockID); err != nil {
		return fmt.Errorf("db: acquire migration advisory lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, migrationAdvisoryLockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("db: create schema_migrations: %w", err)
	}

	applied := map[string]bool{}
	rows, err := conn.Query(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("db: read schema_migrations: %w", err)
	}
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			rows.Close()
			return fmt.Errorf("db: scan schema_migrations: %w", err)
		}
		applied[filename] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("db: read schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("db: read migrations dir: %w", err)
	}

	var filenames []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		filenames = append(filenames, e.Name())
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		if applied[filename] {
			continue
		}

		path := dir + "/" + filename
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("db: read %s: %w", path, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("db: begin tx for %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("db: exec %s: %w", path, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, filename); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("db: record %s as applied: %w", filename, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("db: commit %s: %w", filename, err)
		}
	}

	return nil
}
