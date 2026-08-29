package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

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

	page, err := store.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("got %d assets, want 1 (upsert should not duplicate)", len(page.Items))
	}
	if page.Items[0].Value != "example.com" {
		t.Errorf("asset value = %q, want %q", page.Items[0].Value, "example.com")
	}
	if page.Total != 1 {
		t.Errorf("Total = %d, want 1", page.Total)
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

func TestUserStoreListAndUpdateRoleAndCountAdmins(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewUserStore(pool)

	admin, err := store.Create(ctx, "admin@example.com", "hash", "admin")
	if err != nil {
		t.Fatalf("Create admin returned error: %v", err)
	}
	viewer, err := store.Create(ctx, "viewer@example.com", "hash", "viewer")
	if err != nil {
		t.Fatalf("Create viewer returned error: %v", err)
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List returned %d users, want 2", len(list))
	}

	adminCount, err := store.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins returned error: %v", err)
	}
	if adminCount != 1 {
		t.Fatalf("CountAdmins = %d, want 1", adminCount)
	}

	updated, err := store.UpdateRole(ctx, viewer.ID, "operator")
	if err != nil {
		t.Fatalf("UpdateRole returned error: %v", err)
	}
	if updated.Role != "operator" {
		t.Errorf("UpdateRole result role = %q, want %q", updated.Role, "operator")
	}

	fetched, err := store.GetByID(ctx, viewer.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if fetched.Role != "operator" {
		t.Errorf("GetByID after UpdateRole role = %q, want %q", fetched.Role, "operator")
	}

	if _, err := store.UpdateRole(ctx, admin.ID, "viewer"); err != nil {
		t.Fatalf("UpdateRole on last admin returned error: %v", err)
	}
	adminCount, err = store.CountAdmins(ctx)
	if err != nil {
		t.Fatalf("CountAdmins returned error: %v", err)
	}
	if adminCount != 0 {
		t.Fatalf("CountAdmins after demoting the only admin = %d, want 0 (the handler layer, not the store, enforces keeping one admin)", adminCount)
	}

	randomID := admin.ID
	randomID[0] ^= 0xFF
	if _, err := store.UpdateRole(ctx, randomID, "admin"); err != ErrUserNotFound {
		t.Errorf("UpdateRole for missing user: err = %v, want ErrUserNotFound", err)
	}
}

func TestAgentKeyStoreCreateAndLookup(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAgentKeyStore(pool)

	if err := store.Create(ctx, "sdr-agent-madrid", "hash-of-key-1"); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	agentID, err := store.AgentIDForHash(ctx, "hash-of-key-1")
	if err != nil {
		t.Fatalf("AgentIDForHash returned error: %v", err)
	}
	if agentID != "sdr-agent-madrid" {
		t.Errorf("AgentIDForHash = %q, want %q", agentID, "sdr-agent-madrid")
	}

	if _, err := store.AgentIDForHash(ctx, "hash-of-a-different-key"); err != ErrAgentKeyNotFound {
		t.Errorf("AgentIDForHash for unknown hash: err = %v, want ErrAgentKeyNotFound", err)
	}
}

func TestAgentKeyStoreDuplicateAgentID(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAgentKeyStore(pool)

	if err := store.Create(ctx, "sdr-agent-madrid", "hash-a"); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}
	if err := store.Create(ctx, "sdr-agent-madrid", "hash-b"); err != ErrAgentIDTaken {
		t.Errorf("second Create for same agent_id: err = %v, want ErrAgentIDTaken", err)
	}
}

func TestAgentKeyStoreRevoke(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAgentKeyStore(pool)

	if err := store.Create(ctx, "sdr-agent-madrid", "hash-c"); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if _, err := store.Revoke(ctx, "sdr-agent-madrid"); err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}

	if _, err := store.AgentIDForHash(ctx, "hash-c"); err != ErrAgentKeyNotFound {
		t.Errorf("AgentIDForHash after revoke: err = %v, want ErrAgentKeyNotFound (revoked keys must stop working)", err)
	}

	if _, err := store.Revoke(ctx, "sdr-agent-madrid"); err != ErrAgentKeyNotFound {
		t.Errorf("double Revoke: err = %v, want ErrAgentKeyNotFound", err)
	}
}

func TestRFStoreInsertAndRecentForAgent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewRFStore(pool)

	base := time.Now().Add(-time.Hour)
	for i := 0; i < 3; i++ {
		r := RFReading{
			AgentID:     "sdr-agent-madrid",
			FrequencyHz: 437500000,
			PowerDbm:    -60 - float64(i),
			CapturedAt:  base.Add(time.Duration(i) * time.Minute),
		}
		if err := store.Insert(ctx, &r); err != nil {
			t.Fatalf("Insert returned error: %v", err)
		}
		if r.ID == uuid.Nil {
			t.Error("Insert did not populate the generated ID")
		}
	}

	other := RFReading{AgentID: "sdr-agent-other", FrequencyHz: 437500000, PowerDbm: -50, CapturedAt: base}
	if err := store.Insert(ctx, &other); err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}

	history, err := store.RecentForAgent(ctx, "sdr-agent-madrid", 10)
	if err != nil {
		t.Fatalf("RecentForAgent returned error: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("RecentForAgent returned %d readings, want 3 (should exclude the other agent's reading)", len(history))
	}
	for _, h := range history {
		if h.AgentID != "sdr-agent-madrid" {
			t.Errorf("RecentForAgent returned a reading for agent %q, want only sdr-agent-madrid", h.AgentID)
		}
	}

	all, err := store.Recent(ctx, 10)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("Recent returned %d readings, want 4 (all agents)", len(all))
	}
}

func TestAnomalyStoreInsertAndRecent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	rfStore := NewRFStore(pool)
	anomalyStore := NewAnomalyStore(pool)

	reading := RFReading{AgentID: "sdr-agent-madrid", FrequencyHz: 437500000, PowerDbm: -20, CapturedAt: time.Now()}
	if err := rfStore.Insert(ctx, &reading); err != nil {
		t.Fatalf("Insert (reading) returned error: %v", err)
	}

	a := RFAnomaly{
		ReadingID:      reading.ID,
		AgentID:        reading.AgentID,
		Metric:         "power_dbm",
		Value:          -20,
		BaselineMean:   -60,
		BaselineStddev: 1.5,
		ZScore:         26.6,
	}
	if err := anomalyStore.Insert(ctx, a); err != nil {
		t.Fatalf("Insert (anomaly) returned error: %v", err)
	}

	list, err := anomalyStore.Recent(ctx, 10)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("Recent returned %d anomalies, want 1", len(list))
	}
	if list[0].ReadingID != reading.ID {
		t.Errorf("Recent()[0].ReadingID = %v, want %v", list[0].ReadingID, reading.ID)
	}
	if list[0].Metric != "power_dbm" {
		t.Errorf("Recent()[0].Metric = %q, want %q", list[0].Metric, "power_dbm")
	}
}

func TestAuditStorePagination(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewAuditStore(pool)

	for i := 0; i < 5; i++ {
		if err := store.Insert(ctx, "user", "tester", "test.action", "127.0.0.1", map[string]interface{}{"n": i}); err != nil {
			t.Fatalf("Insert #%d returned error: %v", i, err)
		}
	}

	page1, err := store.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List page 1 returned error: %v", err)
	}
	if page1.Total != 5 {
		t.Errorf("Total = %d, want 5", page1.Total)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page 1 has %d items, want 2", len(page1.Items))
	}

	page2, err := store.List(ctx, 2, 2)
	if err != nil {
		t.Fatalf("List page 2 returned error: %v", err)
	}
	if len(page2.Items) != 2 {
		t.Fatalf("page 2 has %d items, want 2", len(page2.Items))
	}
	if page1.Items[0].ID == page2.Items[0].ID {
		t.Error("page 1 and page 2 returned the same first item; offset was not applied")
	}

	page3, err := store.List(ctx, 2, 4)
	if err != nil {
		t.Fatalf("List page 3 returned error: %v", err)
	}
	if len(page3.Items) != 1 {
		t.Fatalf("page 3 (last, partial) has %d items, want 1", len(page3.Items))
	}

	clamped, err := store.List(ctx, 10000, -5)
	if err != nil {
		t.Fatalf("List with out-of-range params returned error: %v", err)
	}
	if clamped.Limit != MaxPageLimit {
		t.Errorf("Limit = %d, want clamped to MaxPageLimit (%d)", clamped.Limit, MaxPageLimit)
	}
	if clamped.Offset != 0 {
		t.Errorf("Offset = %d, want clamped to 0", clamped.Offset)
	}
}
