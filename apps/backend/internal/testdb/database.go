package testdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/db"
)

func ResetAndMigrate(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	_, err := pool.Exec(ctx, `
		DO $$
		DECLARE
			table_row RECORD;
		BEGIN
			FOR table_row IN
				SELECT tablename
				FROM pg_tables
				WHERE schemaname = current_schema()
			LOOP
				EXECUTE format('DROP TABLE IF EXISTS %I CASCADE', table_row.tablename);
			END LOOP;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("testdb: reset tables: %w", err)
	}

	if err := db.Migrate(ctx, pool, migrationsDir); err != nil {
		return fmt.Errorf("testdb: migrate: %w", err)
	}

	return nil
}
