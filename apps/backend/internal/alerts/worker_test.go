package alerts

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
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
		DROP TABLE IF EXISTS asset_cve_alerts CASCADE;
		DROP TABLE IF EXISTS audit_log CASCADE;
		DROP TABLE IF EXISTS space_cves CASCADE;
		DROP TABLE IF EXISTS assets CASCADE;
		DROP TABLE IF EXISTS programs CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

		CREATE EXTENSION IF NOT EXISTS pgcrypto;

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

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'viewer',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE space_cves (
			cve_id TEXT PRIMARY KEY,
			description TEXT NOT NULL DEFAULT '',
			published TIMESTAMPTZ,
			severity TEXT NOT NULL DEFAULT 'unknown',
			score DOUBLE PRECISION,
			keyword TEXT NOT NULL DEFAULT '',
			synced_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE asset_cve_alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
			cve_id TEXT NOT NULL REFERENCES space_cves(cve_id) ON DELETE CASCADE,
			matched_keyword TEXT NOT NULL,
			severity TEXT NOT NULL,
			acknowledged BOOLEAN NOT NULL DEFAULT false,
			acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
			acknowledged_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (asset_id, cve_id)
		);

		CREATE TABLE audit_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			actor_type TEXT NOT NULL,
			actor_id TEXT NOT NULL,
			action TEXT NOT NULL,
			ip TEXT NOT NULL DEFAULT '',
			details JSONB NOT NULL DEFAULT '{}'::jsonb
		);
	`
	if _, err := pool.Exec(ctx, schema); err != nil {
		t.Fatalf("failed to set up test schema: %v", err)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

func setupWorker(t *testing.T) (*Worker, *pgxpool.Pool) {
	t.Helper()
	pool := testPool(t)
	w := NewWorker(nil, store.NewSpaceCVEStore(pool), store.NewAssetStore(pool), store.NewAssetAlertStore(pool), store.NewAuditStore(pool))
	return w, pool
}

func insertProgram(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO programs (name) VALUES ('test') RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("insert program: %v", err)
	}
	return id
}

func insertUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	email := "test-" + uuid.NewString() + "@example.com"
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id`, email).Scan(&id); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func TestProcessCVECreatesAlertForMatchingAssetKind(t *testing.T) {
	w, pool := setupWorker(t)
	ctx := context.Background()

	programID := insertProgram(t, ctx, pool)
	assets := store.NewAssetStore(pool)
	if err := assets.Upsert(ctx, programID, "satellite", "NORAD-99999"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}
	// A ground_station asset should NOT be matched by a "satellite"-keyword CVE.
	if err := assets.Upsert(ctx, programID, "ground_station", "gs-madrid"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}

	spaceCVEs := store.NewSpaceCVEStore(pool)
	if err := spaceCVEs.Upsert(ctx, "CVE-2026-10001", "test", time.Now(), "HIGH", 7.1, "satellite"); err != nil {
		t.Fatalf("upsert cve: %v", err)
	}

	if err := w.ProcessCVE(ctx, "CVE-2026-10001"); err != nil {
		t.Fatalf("ProcessCVE: %v", err)
	}

	alerts := store.NewAssetAlertStore(pool)
	page, err := alerts.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("got %d alerts, want 1 (only the satellite asset should match)", len(page.Items))
	}
	if page.Items[0].AssetKind != "satellite" {
		t.Errorf("matched asset kind = %q, want satellite", page.Items[0].AssetKind)
	}
	if page.Items[0].CVEID != "CVE-2026-10001" {
		t.Errorf("CVEID = %q, want CVE-2026-10001", page.Items[0].CVEID)
	}
	if page.Items[0].Severity != "HIGH" {
		t.Errorf("Severity = %q, want HIGH", page.Items[0].Severity)
	}
}

func TestProcessCVEWithUnmappedKeywordCreatesNoAlerts(t *testing.T) {
	w, pool := setupWorker(t)
	ctx := context.Background()

	programID := insertProgram(t, ctx, pool)
	assets := store.NewAssetStore(pool)
	if err := assets.Upsert(ctx, programID, "domain", "example.com"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}

	spaceCVEs := store.NewSpaceCVEStore(pool)
	// "domain" assets have no keyword mapping (see matcher.go) — even a
	// CVE that would match other kinds must not alert on this asset.
	if err := spaceCVEs.Upsert(ctx, "CVE-2026-10002", "test", time.Now(), "LOW", 2.0, "satellite"); err != nil {
		t.Fatalf("upsert cve: %v", err)
	}

	if err := w.ProcessCVE(ctx, "CVE-2026-10002"); err != nil {
		t.Fatalf("ProcessCVE: %v", err)
	}

	alerts := store.NewAssetAlertStore(pool)
	page, err := alerts.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("got %d alerts, want 0 (no satellite/ground_station/firmware asset exists)", len(page.Items))
	}
}

func TestProcessCVEIsIdempotent(t *testing.T) {
	w, pool := setupWorker(t)
	ctx := context.Background()

	programID := insertProgram(t, ctx, pool)
	assets := store.NewAssetStore(pool)
	if err := assets.Upsert(ctx, programID, "firmware", "flight-computer-v2"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}

	spaceCVEs := store.NewSpaceCVEStore(pool)
	if err := spaceCVEs.Upsert(ctx, "CVE-2026-10003", "test", time.Now(), "CRITICAL", 9.8, "firmware"); err != nil {
		t.Fatalf("upsert cve: %v", err)
	}

	if err := w.ProcessCVE(ctx, "CVE-2026-10003"); err != nil {
		t.Fatalf("first ProcessCVE: %v", err)
	}
	if err := w.ProcessCVE(ctx, "CVE-2026-10003"); err != nil {
		t.Fatalf("second ProcessCVE: %v", err)
	}

	alerts := store.NewAssetAlertStore(pool)
	page, err := alerts.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("got %d alerts after processing the same CVE twice, want 1 (no duplicates)", len(page.Items))
	}
}

func TestProcessCVEUnknownIDReturnsError(t *testing.T) {
	w, _ := setupWorker(t)
	if err := w.ProcessCVE(context.Background(), "CVE-0000-00000"); err == nil {
		t.Fatal("expected an error for an unknown CVE ID, got nil")
	}
}

func TestProcessCVERecordsAuditEntry(t *testing.T) {
	w, pool := setupWorker(t)
	ctx := context.Background()

	programID := insertProgram(t, ctx, pool)
	assets := store.NewAssetStore(pool)
	if err := assets.Upsert(ctx, programID, "ground_station", "gs-madrid"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}

	spaceCVEs := store.NewSpaceCVEStore(pool)
	if err := spaceCVEs.Upsert(ctx, "CVE-2026-10004", "test", time.Now(), "MEDIUM", 5.0, "RF"); err != nil {
		t.Fatalf("upsert cve: %v", err)
	}

	if err := w.ProcessCVE(ctx, "CVE-2026-10004"); err != nil {
		t.Fatalf("ProcessCVE: %v", err)
	}

	auditLog := store.NewAuditStore(pool)
	page, err := auditLog.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List audit log: %v", err)
	}
	found := false
	for _, e := range page.Items {
		if e.Action == "asset_alert.created" && e.ActorType == "system" {
			found = true
		}
	}
	if !found {
		t.Error("expected an asset_alert.created audit entry from the system actor")
	}
}

func TestAcknowledgeAlert(t *testing.T) {
	w, pool := setupWorker(t)
	ctx := context.Background()

	programID := insertProgram(t, ctx, pool)
	assets := store.NewAssetStore(pool)
	if err := assets.Upsert(ctx, programID, "satellite", "NORAD-88888"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}
	spaceCVEs := store.NewSpaceCVEStore(pool)
	if err := spaceCVEs.Upsert(ctx, "CVE-2026-10005", "test", time.Now(), "LOW", 1.0, "orbit"); err != nil {
		t.Fatalf("upsert cve: %v", err)
	}
	if err := w.ProcessCVE(ctx, "CVE-2026-10005"); err != nil {
		t.Fatalf("ProcessCVE: %v", err)
	}

	alertStore := store.NewAssetAlertStore(pool)
	page, err := alertStore.List(ctx, 0, 0)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("setup: expected exactly 1 alert, got %d (err=%v)", len(page.Items), err)
	}

	userID := insertUser(t, ctx, pool)
	ok, err := alertStore.Acknowledge(ctx, page.Items[0].ID, userID)
	if err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if !ok {
		t.Fatal("Acknowledge returned false for an existing alert")
	}

	page, err = alertStore.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List after acknowledge: %v", err)
	}
	if !page.Items[0].Acknowledged {
		t.Error("alert should be marked acknowledged after Acknowledge")
	}
	if page.Items[0].AcknowledgedBy == nil || *page.Items[0].AcknowledgedBy != userID {
		t.Errorf("AcknowledgedBy = %v, want %v", page.Items[0].AcknowledgedBy, userID)
	}
}

func TestAcknowledgeUnknownAlertReturnsFalse(t *testing.T) {
	_, pool := setupWorker(t)
	alertStore := store.NewAssetAlertStore(pool)

	ok, err := alertStore.Acknowledge(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if ok {
		t.Error("Acknowledge should return false for a non-existent alert ID")
	}
}
