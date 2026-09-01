package main

import (
	"testing"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func TestIncidentsCacheKeyIsTenantIsolated(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	filter := store.IncidentFilter{}

	keyA := incidentsCacheKey(orgA, 50, 0, filter)
	keyB := incidentsCacheKey(orgB, 50, 0, filter)
	if keyA == keyB {
		t.Fatalf("incidents cache key must differ across organizations, got identical key %q for both", keyA)
	}
}

func TestIncidentsCacheKeyDiffersByFilterAndPage(t *testing.T) {
	org := uuid.New()
	base := incidentsCacheKey(org, 50, 0, store.IncidentFilter{})

	cases := []struct {
		name string
		key  string
	}{
		{"different status", incidentsCacheKey(org, 50, 0, store.IncidentFilter{Status: "new"})},
		{"different severity", incidentsCacheKey(org, 50, 0, store.IncidentFilter{Severity: "critical"})},
		{"different agent", incidentsCacheKey(org, 50, 0, store.IncidentFilter{AgentID: "sdr-1"})},
		{"different offset", incidentsCacheKey(org, 50, 25, store.IncidentFilter{})},
		{"different limit", incidentsCacheKey(org, 10, 0, store.IncidentFilter{})},
	}
	for _, c := range cases {
		if c.key == base {
			t.Errorf("%s: expected a distinct cache key from the unfiltered default, got the same key %q", c.name, c.key)
		}
	}
}

func TestIncidentsCacheKeyDefaultMatchesDashboardDefaultQuery(t *testing.T) {
	org := uuid.New()
	want := incidentsCacheKey(org, 50, 0, store.IncidentFilter{})
	got := incidentsCacheKeyDefault(org)
	if got != want {
		t.Fatalf("incidentsCacheKeyDefault() = %q, want %q (limit=50, offset=0, no filter)", got, want)
	}
}

func TestAssetAlertsCacheKeyIsTenantIsolated(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	if assetAlertsCacheKey(orgA, 50, 0) == assetAlertsCacheKey(orgB, 50, 0) {
		t.Fatal("asset-alerts cache key must differ across organizations")
	}
}

func TestAssetAlertsCacheKeyDefaultMatchesDashboardDefaultQuery(t *testing.T) {
	org := uuid.New()
	want := assetAlertsCacheKey(org, 50, 0)
	got := assetAlertsCacheKeyDefault(org)
	if got != want {
		t.Fatalf("assetAlertsCacheKeyDefault() = %q, want %q (limit=50, offset=0)", got, want)
	}
}

func TestRFCacheKeysAreTenantIsolatedAndDistinctFromEachOther(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()

	if rfReadingsCacheKey(orgA) == rfReadingsCacheKey(orgB) {
		t.Fatal("rf readings cache key must differ across organizations")
	}
	if rfAnomaliesCacheKey(orgA) == rfAnomaliesCacheKey(orgB) {
		t.Fatal("rf anomalies cache key must differ across organizations")
	}
	if rfReadingsCacheKey(orgA) == rfAnomaliesCacheKey(orgA) {
		t.Fatal("rf readings and rf anomalies must not share a cache key within the same organization")
	}
}
