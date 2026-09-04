package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mlko01/lyrahcle/apps/backend/internal/sparta"
)

func TestSpartaMatrixEndpointExposesTheCuratedCatalog(t *testing.T) {
	recorder := httptest.NewRecorder()
	handleSpartaMatrix()(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/sparta/matrix", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", recorder.Code)
	}
	var payload struct {
		Framework        string `json:"framework"`
		CatalogVersion   string `json:"catalog_version"`
		CatalogScopeNote string `json:"catalog_scope_note"`
		Tactics          []struct {
			ID string `json:"id"`
		} `json:"tactics"`
		Techniques []struct {
			ID       string `json:"id"`
			TacticID string `json:"tactic_id"`
		} `json:"techniques"`
		Rules []struct {
			EvidenceKind string   `json:"evidence_kind"`
			TechniqueIDs []string `json:"technique_ids"`
			Rationale    string   `json:"rationale"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Framework != sparta.FrameworkName || payload.CatalogVersion != sparta.CatalogVersion {
		t.Fatalf("unexpected framework metadata %+v", payload)
	}
	if payload.CatalogScopeNote == "" {
		t.Fatal("the response must state that the catalog is a curated subset")
	}
	for _, token := range []string{"Manually maintained", "curated subset", "published matrix", "reports", "procurement"} {
		if !strings.Contains(payload.CatalogScopeNote, token) {
			t.Fatalf("catalog scope note must contain %q", token)
		}
	}
	if len(payload.Tactics) != len(sparta.Tactics()) || len(payload.Techniques) != len(sparta.Techniques()) {
		t.Fatalf("unexpected catalog size: %d tactics, %d techniques", len(payload.Tactics), len(payload.Techniques))
	}
	if len(payload.Rules) == 0 {
		t.Fatal("the mapping rules must be published so operators can audit the attribution")
	}
	for _, rule := range payload.Rules {
		if rule.Rationale == "" || len(rule.TechniqueIDs) == 0 {
			t.Fatalf("rule %s is not explainable", rule.EvidenceKind)
		}
	}
}

func TestSpartaCoverageWindowIsBounded(t *testing.T) {
	if spartaWindowDays("") != spartaDefaultWindowDays {
		t.Fatal("an empty window must fall back to the default")
	}
	if spartaWindowDays("-4") != spartaDefaultWindowDays || spartaWindowDays("abc") != spartaDefaultWindowDays {
		t.Fatal("invalid windows must fall back to the default")
	}
	if spartaWindowDays("5000") != spartaMaxEvidenceWindowDays {
		t.Fatal("the window must be capped")
	}
	if spartaWindowDays("7") != 7 {
		t.Fatal("a valid window must be honoured")
	}
}

func TestSpartaAndExternalObservationRoutesAreCapabilityGated(t *testing.T) {
	required := map[string]string{
		"routes_sparta.go":               `a.perm("detections.read")).Get("/matrix"`,
		"routes_satellites_tmtc.go":      `a.perm("orbital.manage")).Post("/{satelliteID}/external-observations/import"`,
		"routes_satellites_tmtc.go.read": `a.perm("orbital.read")).Get("/{satelliteID}/external-observations"`,
		"routes_sparta.go.coverage":      `a.perm("detections.read")).Get("/coverage"`,
	}
	for name, token := range required {
		file := strings.Split(name, ".go")[0] + ".go"
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if !strings.Contains(string(data), token) {
			t.Fatalf("%s must gate the route with %s", file, token)
		}
	}
}

func TestExternalObservationsAreStoredAsContextOnlyEvidence(t *testing.T) {
	handlers, err := os.ReadFile("routes_external_observations.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(handlers)
	for _, token := range []string{
		"satnogs.EvidenceKindExternal",
		"satnogs.ApplicabilityReception",
		"does not prove onboard health",
		"audit.ActionExternalReceptionImported",
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("external observation handlers must keep %q", token)
		}
	}
	if strings.Contains(source, "rf_readings") || strings.Contains(source, "AnomalyStore") {
		t.Fatal("third-party reception must never be written into measured RF evidence")
	}

	migration, err := os.ReadFile("../../migrations/0046_external_rf_observations.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(migration)
	for _, token := range []string{
		"CHECK (confidence BETWEEN 0 AND 60)",
		"lyrahcle_rls_allows_organization(organization_id)",
		"FORCE ROW LEVEL SECURITY",
		"reception_context_only",
	} {
		if !strings.Contains(schema, token) {
			t.Fatalf("migration must keep %q", token)
		}
	}
}
