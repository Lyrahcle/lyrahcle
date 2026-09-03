package alerts

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/testdb"
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
		pool.Close()
		t.Fatalf("failed to ping test database: %v", err)
	}
	if err := testdb.ResetAndMigrate(ctx, pool, "../../migrations"); err != nil {
		pool.Close()
		t.Fatalf("failed to prepare test database: %v", err)
	}

	t.Cleanup(pool.Close)
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
	email := "test-" + uuid.NewString() + "@example.com"
	user, err := store.NewUserStore(pool).Create(ctx, email, "x", "viewer")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return user.ID
}

func TestProcessCVECreatesAlertForMatchingAssetKind(t *testing.T) {
	w, pool := setupWorker(t)
	ctx := context.Background()

	programID := insertProgram(t, ctx, pool)
	assets := store.NewAssetStore(pool)
	if err := assets.Upsert(ctx, programID, "satellite", "NORAD-99999"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}

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
	if !page.Items[0].ContextOnly || page.Items[0].ApplicabilityStatus != "unknown" || page.Items[0].MatchMethod != "thematic_asset_kind" || page.Items[0].Confidence != 20 {
		t.Errorf("assessment = context_only:%v status:%q method:%q confidence:%d", page.Items[0].ContextOnly, page.Items[0].ApplicabilityStatus, page.Items[0].MatchMethod, page.Items[0].Confidence)
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

func TestProcessCVERetryDoesNotDuplicateAudit(t *testing.T) {
	w, pool := setupWorker(t)
	ctx := context.Background()

	programID := insertProgram(t, ctx, pool)
	assets := store.NewAssetStore(pool)
	if err := assets.Upsert(ctx, programID, "satellite", "NORAD-77777"); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}
	spaceCVEs := store.NewSpaceCVEStore(pool)
	if err := spaceCVEs.Upsert(ctx, "CVE-2026-10006", "test", time.Now(), "HIGH", 8.0, "satellite"); err != nil {
		t.Fatalf("upsert cve: %v", err)
	}

	if err := w.ProcessCVE(ctx, "CVE-2026-10006"); err != nil {
		t.Fatalf("first ProcessCVE: %v", err)
	}
	if err := w.ProcessCVE(ctx, "CVE-2026-10006"); err != nil {
		t.Fatalf("retry ProcessCVE: %v", err)
	}

	var alertsCount, auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM asset_cve_alerts WHERE cve_id='CVE-2026-10006'`).Scan(&alertsCount); err != nil {
		t.Fatalf("count alerts: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action='asset_alert.created' AND details->>'cve_id'='CVE-2026-10006'`).Scan(&auditCount); err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if alertsCount != 1 || auditCount != 1 {
		t.Fatalf("retry counts: alerts=%d audit=%d, want 1/1", alertsCount, auditCount)
	}
}
