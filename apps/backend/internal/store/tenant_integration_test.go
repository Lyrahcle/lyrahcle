package store

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTenantIsolationSnapshotIntegration(t *testing.T) {
	dsn := os.Getenv("LYRAHCLE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set LYRAHCLE_TEST_DATABASE_URL to run database-backed tenant isolation tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	scope := NewTenantScopeStore(pool)
	orgA := uuid.New()
	orgB := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO organizations (id,slug,name) VALUES
		($1,$2,$3),($4,$5,$6)`,
		orgA, "tenant-a-"+orgA.String()[:8], "Tenant A",
		orgB, "tenant-b-"+orgB.String()[:8], "Tenant B")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM organizations WHERE id=ANY($1)`, []uuid.UUID{orgA, orgB})
	})

	_, err = pool.Exec(ctx, `INSERT INTO programs (organization_id,name) VALUES ($1,'A-program'),($2,'B-program')`, orgA, orgB)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM programs WHERE organization_id=ANY($1)`, []uuid.UUID{orgA, orgB})
	})

	a, err := scope.Snapshot(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	b, err := scope.Snapshot(ctx, orgB)
	if err != nil {
		t.Fatal(err)
	}
	if a.Programs != 1 || b.Programs != 1 {
		t.Fatalf("tenant snapshots leaked or missed rows: A=%d B=%d", a.Programs, b.Programs)
	}
}
