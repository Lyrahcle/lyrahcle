package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

const securityE2EPasswordV1463 = "Correct-Horse-Battery-9!"

func securityE2EOrganizationV1463(t *testing.T, pool *pgxpool.Pool, label string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	slug := fmt.Sprintf("security-%s-%s", label, id.String()[:8])
	if _, err := pool.Exec(context.Background(), `
        INSERT INTO organizations (id, slug, name)
        VALUES ($1, $2, $3)
    `, id, slug, "Security E2E "+label); err != nil {
		t.Fatalf("create organization %s: %v", label, err)
	}
	return id
}

func securityE2EUserV1463(t *testing.T, router http.Handler, users *store.UserStore, orgID uuid.UUID, email string, role auth.Role) (store.User, string) {
	t.Helper()
	hash, err := auth.HashPassword(securityE2EPasswordV1463)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := users.CreateInOrganization(context.Background(), orgID, email, hash, string(role))
	if err != nil {
		t.Fatalf("create %s: %v", email, err)
	}
	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": email,
		"password":   securityE2EPasswordV1463,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: status=%d body=%s", email, rec.Code, rec.Body.String())
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login for %s: %v", email, err)
	}
	if body.Token == "" {
		t.Fatalf("login %s returned empty token", email)
	}
	return user, body.Token
}

func TestSecurityE2EHTTPMultiTenantBoundariesV1463(t *testing.T) {
	router, pool, users := testRouter(t)
	ctx := context.Background()

	orgA := securityE2EOrganizationV1463(t, pool, "alpha")
	orgB := securityE2EOrganizationV1463(t, pool, "bravo")

	adminA, adminAToken := securityE2EUserV1463(t, router, users, orgA, "admin-a-v1463@example.com", auth.RoleAdmin)
	_, viewerAToken := securityE2EUserV1463(t, router, users, orgA, "viewer-a-v1463@example.com", auth.RoleViewer)
	adminB, adminBToken := securityE2EUserV1463(t, router, users, orgB, "admin-b-v1463@example.com", auth.RoleAdmin)

	programs := store.NewProgramStore(pool)
	assets := store.NewAssetStore(pool)
	incidents := store.NewIncidentStore(pool)
	rf := store.NewRFStore(pool)

	programA, err := programs.CreateInOrganization(ctx, orgA, "Alpha program")
	if err != nil {
		t.Fatal(err)
	}
	programB, err := programs.CreateInOrganization(ctx, orgB, "Bravo program")
	if err != nil {
		t.Fatal(err)
	}
	if err := assets.UpsertInOrganization(ctx, orgA, programA.ID, "domain", "alpha-v1463.example"); err != nil {
		t.Fatal(err)
	}
	if err := assets.UpsertInOrganization(ctx, orgB, programB.ID, "domain", "bravo-v1463.example"); err != nil {
		t.Fatal(err)
	}
	pageB, err := assets.ListByOrganization(ctx, orgB, 10, 0)
	if err != nil || len(pageB.Items) != 1 {
		t.Fatalf("seed/list tenant B asset: items=%d err=%v", len(pageB.Items), err)
	}
	assetB := pageB.Items[0]

	incidentB, err := incidents.Create(ctx, store.CreateIncidentInput{
		OrganizationID: orgB,
		Title:          "Tenant B incident",
		Description:    "Must remain invisible to tenant A",
		Severity:       "high",
		SourceType:     "manual",
		CreatedBy:      &adminB.ID,
	})
	if err != nil {
		t.Fatalf("seed tenant B incident: %v", err)
	}
	readingA := store.RFReading{OrganizationID: orgA, AgentID: "agent-alpha-v1463", FrequencyHz: 145800000, PowerDbm: -82, CapturedAt: time.Now().UTC()}
	readingB := store.RFReading{OrganizationID: orgB, AgentID: "agent-bravo-v1463", FrequencyHz: 437500000, PowerDbm: -61, CapturedAt: time.Now().UTC()}
	if err := rf.Insert(ctx, &readingA); err != nil {
		t.Fatalf("seed tenant A RF reading: %v", err)
	}
	if err := rf.Insert(ctx, &readingB); err != nil {
		t.Fatalf("seed tenant B RF reading: %v", err)
	}

	t.Run("authentication is required before tenant data", func(t *testing.T) {
		rec := doJSON(t, router, http.MethodGet, "/api/v1/assets/", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("missing bearer token status=%d want=401 body=%s", rec.Code, rec.Body.String())
		}
		rec = doJSON(t, router, http.MethodGet, "/api/v1/assets/", "not-a-valid-jwt", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("garbage bearer token status=%d want=401 body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list endpoints expose only caller tenant", func(t *testing.T) {
		rec := doJSON(t, router, http.MethodGet, "/api/v1/assets/?limit=50&offset=0", adminAToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("tenant A list assets: status=%d body=%s", rec.Code, rec.Body.String())
		}
		var page store.Page[store.Asset]
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 1 || page.Items[0].Value != "alpha-v1463.example" || page.Items[0].OrganizationID != orgA {
			t.Fatalf("tenant A asset list leaked or missed data: %#v", page.Items)
		}

		rec = doJSON(t, router, http.MethodGet, "/api/v1/rf/readings", adminAToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("tenant A list RF readings: status=%d body=%s", rec.Code, rec.Body.String())
		}
		var readings []store.RFReading
		if err := json.Unmarshal(rec.Body.Bytes(), &readings); err != nil {
			t.Fatal(err)
		}
		if len(readings) != 1 || readings[0].OrganizationID != orgA || readings[0].AgentID != readingA.AgentID {
			t.Fatalf("tenant A RF list leaked or missed data: %#v", readings)
		}
	})

	t.Run("valid foreign UUIDs cannot be read or mutated", func(t *testing.T) {
		rec := doJSON(t, router, http.MethodPut, "/api/v1/assets/"+assetB.ID.String(), adminAToken, map[string]string{
			"program_id": programA.ID.String(),
			"kind":       "domain",
			"value":      "idor-mutated-v1463.example",
		})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant asset update status=%d want=404 body=%s", rec.Code, rec.Body.String())
		}
		rec = doJSON(t, router, http.MethodDelete, "/api/v1/assets/"+assetB.ID.String(), adminAToken, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant asset delete status=%d want=404 body=%s", rec.Code, rec.Body.String())
		}
		rec = doJSON(t, router, http.MethodGet, "/api/v1/incidents/"+incidentB.ID.String(), adminAToken, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant incident read status=%d want=404 body=%s", rec.Code, rec.Body.String())
		}
		rec = doJSON(t, router, http.MethodPatch, "/api/v1/incidents/"+incidentB.ID.String(), adminAToken, map[string]string{"status": "investigating"})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant incident update status=%d want=404 body=%s", rec.Code, rec.Body.String())
		}
		rec = doJSON(t, router, http.MethodPatch, "/api/v1/users/"+adminB.ID.String()+"/role", adminAToken, map[string]string{"role": "viewer"})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant user role update status=%d want=404 body=%s", rec.Code, rec.Body.String())
		}

		if ok, err := assets.ExistsInOrganization(ctx, orgB, assetB.ID); err != nil {
			t.Fatal(err)
		} else if !ok {
			t.Fatal("cross-tenant delete removed tenant B asset")
		}

		refreshedB, err := users.GetByIDInOrganization(ctx, orgB, adminB.ID)
		if err != nil {
			t.Fatal(err)
		}
		if refreshedB.Role != string(auth.RoleAdmin) {
			t.Fatalf("cross-tenant mutation changed tenant B role to %q", refreshedB.Role)
		}
		detailB, err := incidents.GetByOrganization(ctx, orgB, incidentB.ID)
		if err != nil {
			t.Fatal(err)
		}
		if detailB.Incident.Status != "new" {
			t.Fatalf("cross-tenant mutation changed tenant B incident status to %q", detailB.Incident.Status)
		}
	})

	t.Run("foreign parent references are rejected", func(t *testing.T) {
		rec := doJSON(t, router, http.MethodPost, "/api/v1/assets/", adminAToken, map[string]string{
			"program_id": programB.ID.String(),
			"kind":       "domain",
			"value":      "cross-parent-v1463.example",
		})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant program reference status=%d want=404 body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("viewer read permission does not imply manage", func(t *testing.T) {
		rec := doJSON(t, router, http.MethodGet, "/api/v1/assets/", viewerAToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("viewer read assets status=%d want=200 body=%s", rec.Code, rec.Body.String())
		}
		rec = doJSON(t, router, http.MethodPost, "/api/v1/programs/", viewerAToken, map[string]string{"name": "Denied program"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("viewer create program status=%d want=403 body=%s", rec.Code, rec.Body.String())
		}
		rec = doJSON(t, router, http.MethodPost, "/api/v1/users/", viewerAToken, map[string]string{
			"email": "denied-v1463@example.com", "password": securityE2EPasswordV1463, "role": "viewer",
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("viewer create user status=%d want=403 body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("organization claim substitution fails closed", func(t *testing.T) {
		forged, _, err := auth.IssueOrganizationUserTokenWithState(
			[]byte("test-jwt-secret-for-http-tests-only"), adminA.ID, orgB, auth.RoleAdmin, false, adminA.AuthVersion, time.Hour,
		)
		if err != nil {
			t.Fatal(err)
		}
		rec := doJSON(t, router, http.MethodGet, "/api/v1/assets/", forged, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("forged tenant claim status=%d want=401 body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("auth version change revokes existing token", func(t *testing.T) {
		if _, err := users.UpdateRoleInOrganization(ctx, orgB, adminB.ID, string(auth.RoleOperator)); err != nil {
			t.Fatal(err)
		}
		rec := doJSON(t, router, http.MethodGet, "/api/v1/assets/", adminBToken, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("stale auth-version token status=%d want=401 body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestSecurityE2ELoginAuditIsTenantScopedV1463(t *testing.T) {
	router, pool, users := testRouter(t)
	ctx := context.Background()

	orgA := securityE2EOrganizationV1463(t, pool, "audit-alpha")
	orgB := securityE2EOrganizationV1463(t, pool, "audit-bravo")
	_, tokenA := securityE2EUserV1463(t, router, users, orgA, "audit-a-v1463@example.com", auth.RoleAdmin)
	userB, _ := securityE2EUserV1463(t, router, users, orgB, "audit-b-v1463@example.com", auth.RoleAdmin)

	var orgBLogins int
	if err := pool.QueryRow(ctx, `
        SELECT count(*) FROM audit_log
        WHERE organization_id=$1 AND actor_id=$2 AND action=$3
    `, orgB, userB.ID.String(), audit.ActionLogin).Scan(&orgBLogins); err != nil {
		t.Fatal(err)
	}
	if orgBLogins != 1 {
		t.Fatalf("tenant B login audit count=%d want=1", orgBLogins)
	}

	rec := doJSON(t, router, http.MethodGet, "/api/v1/audit-log?limit=100&offset=0", tokenA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant A audit-log status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page store.Page[store.AuditEntry]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	for _, entry := range page.Items {
		if entry.OrganizationID != orgA {
			t.Fatalf("tenant A audit endpoint leaked organization %s entry %#v", entry.OrganizationID, entry)
		}
		if entry.ActorID == userB.ID.String() {
			t.Fatalf("tenant A audit endpoint leaked tenant B login entry %#v", entry)
		}
	}
}
