package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestSecurityE2EMultiTenantStoreBoundaries(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	orgA := uuid.New()
	orgB := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (id, slug, name)
		VALUES ($1,$2,$3),($4,$5,$6)
	`, orgA, "security-a-"+orgA.String()[:8], "Security Tenant A",
		orgB, "security-b-"+orgB.String()[:8], "Security Tenant B"); err != nil {
		t.Fatal(err)
	}

	programs := NewProgramStore(pool)
	assets := NewAssetStore(pool)
	rules := NewDetectionRuleStore(pool)
	incidents := NewIncidentStore(pool)

	programA, err := programs.CreateInOrganization(ctx, orgA, "A-program")
	if err != nil {
		t.Fatal(err)
	}
	programB, err := programs.CreateInOrganization(ctx, orgB, "B-program")
	if err != nil {
		t.Fatal(err)
	}

	if err := assets.UpsertInOrganization(ctx, orgA, programA.ID, "domain", "a.example"); err != nil {
		t.Fatal(err)
	}
	if err := assets.UpsertInOrganization(ctx, orgB, programB.ID, "domain", "b.example"); err != nil {
		t.Fatal(err)
	}

	pageA, err := assets.ListByOrganization(ctx, orgA, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	pageB, err := assets.ListByOrganization(ctx, orgB, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pageA.Items) != 1 || pageA.Items[0].Value != "a.example" {
		t.Fatalf("tenant A asset list leaked or missed data: %#v", pageA.Items)
	}
	if len(pageB.Items) != 1 || pageB.Items[0].Value != "b.example" {
		t.Fatalf("tenant B asset list leaked or missed data: %#v", pageB.Items)
	}
	if ok, err := assets.ExistsInOrganization(ctx, orgA, pageB.Items[0].ID); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("tenant A can resolve tenant B asset by id")
	}

	var ruleBID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO rf_detection_rules (
			organization_id, name, description, enabled, metric, operator,
			threshold, severity, cooldown_seconds, auto_incident
		) VALUES ($1,$2,'',true,'power_dbm','abs_z_gte',6,'high',120,true)
		RETURNING id
	`, orgB, "tenant-b-rule-"+orgB.String()[:8]).Scan(&ruleBID); err != nil {
		t.Fatal(err)
	}

	if _, err := rules.GetByOrganization(ctx, orgA, ruleBID); !errors.Is(err, ErrDetectionRuleNotFound) {
		t.Fatalf("tenant A resolved tenant B detection rule: err=%v", err)
	}

	var incidentBID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO incidents (organization_id, reference, title, severity)
		VALUES ($1,$2,$3,'high')
		RETURNING id
	`, orgB, "SEC-"+orgB.String()[:8], "Tenant B incident").Scan(&incidentBID); err != nil {
		t.Fatal(err)
	}

	if _, err := incidents.GetByOrganization(ctx, orgA, incidentBID); !errors.Is(err, ErrIncidentNotFound) {
		t.Fatalf("tenant A resolved tenant B incident: err=%v", err)
	}
	if _, err := incidents.UpdateByOrganization(ctx, orgA, incidentBID, UpdateIncidentInput{
		Status: "investigating",
	}); !errors.Is(err, ErrIncidentNotFound) {
		t.Fatalf("tenant A mutated tenant B incident: err=%v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM incidents WHERE id=$1`, incidentBID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "new" {
		t.Fatalf("cross-tenant mutation changed tenant B incident status to %q", status)
	}
}
