package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
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
	if page.Items[0].OrganizationID != DefaultOrganizationID {
		t.Errorf("asset organization = %v, want default organization %v", page.Items[0].OrganizationID, DefaultOrganizationID)
	}
	byKind, err := store.ListByKinds(ctx, []string{"domain"})
	if err != nil {
		t.Fatalf("ListByKinds returned error: %v", err)
	}
	if len(byKind) != 1 || byKind[0].OrganizationID != DefaultOrganizationID {
		t.Fatalf("ListByKinds = %+v, want one default-organization asset", byKind)
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
	if satellites[0].TLEEpoch == nil {
		t.Fatal("TLE epoch was not persisted")
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

func TestIncidentStoreLifecycle(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserStore(pool)
	incidents := NewIncidentStore(pool)

	actor, err := users.Create(ctx, "incident-admin@example.com", "test-password-hash", "admin")
	if err != nil {
		t.Fatalf("Create user returned error: %v", err)
	}

	incident, err := incidents.Create(ctx, CreateIncidentInput{
		Title:       "Unexpected RF behavior",
		Description: "Investigate the signal deviation.",
		Severity:    "high",
		SourceType:  "manual",
		AgentID:     "sdr-store-test",
		AssignedTo:  &actor.ID,
		CreatedBy:   &actor.ID,
	})
	if err != nil {
		t.Fatalf("Create incident returned error: %v", err)
	}
	if incident.ID == uuid.Nil || incident.Reference == "" {
		t.Fatalf("created incident missing identity: %+v", incident)
	}

	page, err := incidents.List(ctx, 50, 0, IncidentFilter{Severity: "high", AgentID: "sdr-store-test"})
	if err != nil {
		t.Fatalf("List incidents returned error: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("incident page = total %d, items %d; want 1, 1", page.Total, len(page.Items))
	}

	updated, err := incidents.Update(ctx, incident.ID, UpdateIncidentInput{
		Status:  "investigating",
		ActorID: actor.ID,
	})
	if err != nil {
		t.Fatalf("Update incident returned error: %v", err)
	}
	if updated.Status != "investigating" {
		t.Fatalf("updated status = %q, want investigating", updated.Status)
	}

	if err := incidents.AddNote(ctx, incident.ID, actor.ID, "Validated against the RF baseline."); err != nil {
		t.Fatalf("AddNote returned error: %v", err)
	}

	detail, err := incidents.Get(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Get incident returned error: %v", err)
	}
	if len(detail.Events) != 3 {
		t.Fatalf("timeline has %d events, want 3", len(detail.Events))
	}
	if detail.Events[0].EventType != "created" || detail.Events[1].EventType != "status_changed" || detail.Events[2].EventType != "note" {
		t.Fatalf("unexpected event sequence: %+v", detail.Events)
	}
}

func TestIncidentStoreRejectsInvalidStatusTransition(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserStore(pool)
	incidents := NewIncidentStore(pool)
	actor, err := users.Create(ctx, "incident-state@example.com", "hash", "operator")
	if err != nil {
		t.Fatal(err)
	}
	incident, err := incidents.Create(ctx, CreateIncidentInput{Title: "State", Severity: "high", SourceType: "manual", CreatedBy: &actor.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"investigating", "resolved"} {
		incident, err = incidents.Update(ctx, incident.ID, UpdateIncidentInput{Status: status, ActorID: actor.ID})
		if err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}
	_, err = incidents.Update(ctx, incident.ID, UpdateIncidentInput{Status: "new", ActorID: actor.ID})
	if !errors.Is(err, ErrInvalidIncidentTransition) {
		t.Fatalf("err = %v, want ErrInvalidIncidentTransition", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE incidents SET status = 'new' WHERE id = $1`, incident.ID); err == nil {
		t.Fatal("database accepted invalid resolved -> new transition")
	}
}

func TestDetectionRuleMatchCooldown(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserStore(pool)
	actor, err := users.Create(ctx, "rule-store@example.com", "test-hash", "admin")
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	rules := NewDetectionRuleStore(pool)
	organizationID := actor.OrganizationID
	rule, err := rules.CreateInOrganization(ctx, organizationID, detection.Rule{
		Name: "Cooldown integration rule", Enabled: true, Metric: "power_dbm", Operator: "abs_z_gte",
		Threshold: 6, Severity: "high", CooldownSeconds: 300, AutoIncident: true,
	}, actor.ID)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	rf := NewRFStore(pool)
	anomalies := NewAnomalyStore(pool)
	newAnomaly := func(power float64) RFAnomaly {
		reading := RFReading{OrganizationID: organizationID, AgentID: "cooldown-agent", FrequencyHz: 437500000, PowerDbm: power, CapturedAt: time.Now().UTC()}
		if err := rf.Insert(ctx, &reading); err != nil {
			t.Fatalf("insert reading: %v", err)
		}
		anomaly, err := anomalies.InsertReturning(ctx, RFAnomaly{
			OrganizationID: organizationID, ReadingID: reading.ID, AgentID: reading.AgentID, Metric: "power_dbm", Value: power,
			BaselineMean: -60, BaselineStddev: 2, ZScore: 9,
		})
		if err != nil {
			t.Fatalf("insert anomaly: %v", err)
		}
		return anomaly
	}

	first := newAnomaly(-10)
	if _, created, err := rules.RecordMatchIfAllowed(ctx, organizationID, rule, first.ID, first.AgentID, first.Metric, first.ZScore); err != nil || !created {
		t.Fatalf("first match: created=%v err=%v", created, err)
	}
	second := newAnomaly(-9)
	if _, created, err := rules.RecordMatchIfAllowed(ctx, organizationID, rule, second.ID, second.AgentID, second.Metric, second.ZScore); err != nil || created {
		t.Fatalf("second match inside cooldown: created=%v err=%v", created, err)
	}
}

func TestUserStoreLifecycle(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewUserStore(pool)

	user, err := store.CreateProvisioned(ctx, "lifecycle@example.com", "hash", "analyst")
	if err != nil {
		t.Fatalf("CreateProvisioned: %v", err)
	}
	if !user.Enabled || !user.MustChangePassword {
		t.Fatalf("provisioned user state = enabled:%v must_change:%v", user.Enabled, user.MustChangePassword)
	}
	if _, err := store.SetEnabled(ctx, user.ID, false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	fetched, err := store.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if fetched.Enabled {
		t.Fatal("disabled user returned enabled=true")
	}
	if err := store.SoftDelete(ctx, user.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if _, err := store.GetByID(ctx, user.ID); err != ErrUserNotFound {
		t.Fatalf("GetByID after delete: err = %v, want ErrUserNotFound", err)
	}
}

func TestWebAuthnHandleAndCredentialCount(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserStore(pool)
	credentials := NewWebAuthnStore(pool)

	user, err := users.Create(ctx, "webauthn@example.com", "hash", "viewer")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	handle, err := credentials.Handle(ctx, user.ID)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(handle) != 32 {
		t.Fatalf("handle length = %d, want 32", len(handle))
	}
	count, err := credentials.Count(ctx, user.ID)
	if err != nil {
		t.Fatalf("Count returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("credential count = %d, want 0", count)
	}
}

func TestPasswordResetTokenIsSingleUse(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserStore(pool)
	resets := NewPasswordResetStore(pool)

	user, err := users.Create(ctx, "reset@example.com", "old-hash", "viewer")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	tokenHash := []byte("01234567890123456789012345678901")
	if err := resets.Create(ctx, user.ID, tokenHash, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("Create reset token returned error: %v", err)
	}
	resolved, err := resets.Lookup(ctx, tokenHash)
	if err != nil || resolved != user.ID {
		t.Fatalf("Lookup = %v, %v", resolved, err)
	}
	if _, err := resets.ConsumeAndUpdatePassword(ctx, tokenHash, "new-hash"); err != nil {
		t.Fatalf("ConsumeAndUpdatePassword returned error: %v", err)
	}
	if _, err := resets.Lookup(ctx, tokenHash); err == nil {
		t.Fatal("used reset token remained valid")
	}
	updated, err := users.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if updated.PasswordHash != "new-hash" {
		t.Fatalf("password hash = %q, want new-hash", updated.PasswordHash)
	}
	if updated.AuthVersion <= user.AuthVersion {
		t.Fatalf("auth version = %d, want > %d", updated.AuthVersion, user.AuthVersion)
	}
}

func TestGlobalSearchFindsSatelliteAndCVE(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	search := NewSearchStore(pool)

	if _, err := pool.Exec(ctx, `INSERT INTO satellites (norad_id, name, tle_line1, tle_line2) VALUES (25544, 'ISS SEARCH', 'L1', 'L2')`); err != nil {
		t.Fatalf("insert satellite: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO space_cves (cve_id, description, severity, keyword) VALUES ('CVE-2099-1234', 'satellite search test', 'high', 'satellite')`); err != nil {
		t.Fatalf("insert CVE: %v", err)
	}

	satResults, err := search.Search(ctx, "ISS SEARCH", 12)
	if err != nil {
		t.Fatalf("satellite Search returned error: %v", err)
	}
	if len(satResults) == 0 || satResults[0].Type != "satellites" {
		t.Fatalf("satellite results = %#v", satResults)
	}

	cveResults, err := search.Search(ctx, "CVE-2099-1234", 12)
	if err != nil {
		t.Fatalf("CVE Search returned error: %v", err)
	}
	found := false
	for _, result := range cveResults {
		if result.Type == "cves" && result.ID == "CVE-2099-1234" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("CVE result not found: %#v", cveResults)
	}
}

func TestAgentIdentityEnrollmentAndQuarantine(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	identities := NewAgentIdentityStore(pool)

	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	if err := identities.CreateEnrollmentToken(ctx, "sdr-agent-secure", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", expiresAt); err != nil {
		t.Fatalf("CreateEnrollmentToken returned error: %v", err)
	}

	agentID, err := identities.AgentIDForEnrollmentToken(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("AgentIDForEnrollmentToken returned error: %v", err)
	}
	if agentID != "sdr-agent-secure" {
		t.Fatalf("agent id = %q, want sdr-agent-secure", agentID)
	}

	cert := AgentCertificate{
		SerialNumber:      "deadbeef",
		FingerprintSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CertificatePEM:    "test-certificate",
		NotBefore:         time.Now().UTC().Add(-time.Minute),
		NotAfter:          time.Now().UTC().Add(time.Hour),
	}
	consumedAgentID, err := identities.ConsumeEnrollmentToken(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", cert, "v9.0", "fw-1")
	if err != nil {
		t.Fatalf("ConsumeEnrollmentToken returned error: %v", err)
	}
	if consumedAgentID != "sdr-agent-secure" {
		t.Fatalf("consumed agent id = %q", consumedAgentID)
	}
	if _, err := identities.AgentIDForEnrollmentToken(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); !errors.Is(err, ErrAgentEnrollmentInvalid) {
		t.Fatalf("reused enrollment token err = %v, want ErrAgentEnrollmentInvalid", err)
	}

	resolved, err := identities.AgentIDForCertificate(ctx, cert.FingerprintSHA256)
	if err != nil {
		t.Fatalf("AgentIDForCertificate returned error: %v", err)
	}
	if resolved != "sdr-agent-secure" {
		t.Fatalf("resolved agent id = %q", resolved)
	}

	if err := identities.SetStatus(ctx, "sdr-agent-secure", "quarantined"); err != nil {
		t.Fatalf("SetStatus quarantined returned error: %v", err)
	}
	if _, err := identities.AgentIDForCertificate(ctx, cert.FingerprintSHA256); !errors.Is(err, ErrAgentCertificateNotFound) {
		t.Fatalf("quarantined certificate err = %v, want ErrAgentCertificateNotFound", err)
	}
	if err := identities.SetStatus(ctx, "sdr-agent-secure", "active"); err != nil {
		t.Fatalf("SetStatus active returned error: %v", err)
	}
	if _, err := identities.AgentIDForCertificate(ctx, cert.FingerprintSHA256); err != nil {
		t.Fatalf("reactivated certificate returned error: %v", err)
	}
}

func TestAgentIdentityRevocationIsIrreversible(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	identities := NewAgentIdentityStore(pool)
	if err := identities.CreateEnrollmentToken(ctx, "sdr-agent-revoked", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", time.Now().Add(15*time.Minute)); err != nil {
		t.Fatal(err)
	}
	cert := AgentCertificate{
		SerialNumber:      "cafebabe",
		FingerprintSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		CertificatePEM:    "test-certificate",
		NotBefore:         time.Now().Add(-time.Minute),
		NotAfter:          time.Now().Add(time.Hour),
	}
	if _, err := identities.ConsumeEnrollmentToken(ctx, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", cert, "v9.0", ""); err != nil {
		t.Fatal(err)
	}
	if err := identities.SetStatus(ctx, "sdr-agent-revoked", "revoked"); err != nil {
		t.Fatal(err)
	}
	if err := identities.SetStatus(ctx, "sdr-agent-revoked", "active"); !errors.Is(err, ErrAgentStatusTransition) {
		t.Fatalf("reactivating revoked agent err = %v, want ErrAgentStatusTransition", err)
	}
}

func TestOrbitalEventStoreCreateAndRecentForSatellite(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	satellites := NewSatelliteStore(pool)
	events := NewOrbitalEventStore(pool)

	line1 := "1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995"
	line2 := "2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754"
	if err := satellites.Upsert(ctx, 25544, "ISS (ZARYA)", line1, line2); err != nil {
		t.Fatalf("satellite upsert: %v", err)
	}
	list, err := satellites.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("satellite list: len=%d err=%v", len(list), err)
	}

	eventTime := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	created, err := events.Create(ctx, CreateOrbitalEventInput{
		SatelliteID: list[0].ID, EventType: "conjunction", EventTime: eventTime,
		Source: "test-feed", Confidence: 0.8, Severity: "medium",
		Metadata: map[string]interface{}{"miss_distance_km": 12.5},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.EventType != "conjunction" || created.Source != "test-feed" {
		t.Fatalf("unexpected created event: %#v", created)
	}

	recent, err := events.RecentForSatellite(ctx, list[0].ID, time.Now().UTC().Add(-time.Hour), 10)
	if err != nil {
		t.Fatalf("RecentForSatellite returned error: %v", err)
	}
	if len(recent) != 1 || recent[0].ID != created.ID {
		t.Fatalf("recent events = %#v", recent)
	}
}
