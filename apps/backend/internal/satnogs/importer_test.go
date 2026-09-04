package satnogs

import (
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/orbit"
)

const (
	issLine1 = "1 25544U 98067A   24200.50000000  .00016717  00000+0  30182-3 0  9991"
	issLine2 = "2 25544  51.6400 100.0000 0006000  60.0000  40.0000 15.50000000123456"
)

func downlink(value int64) *int64 { return &value }

func coordinate(value float64) *float64 { return &value }

func TestBuildObservationRecordsFiltersUnusableEntries(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	observations := []Observation{
		{ID: 1, Start: now.Add(-2 * time.Hour), End: now.Add(-2*time.Hour + 10*time.Minute), Status: "good", VettedStatus: "good"},
		{ID: 1, Start: now.Add(-3 * time.Hour), End: now.Add(-3*time.Hour + 5*time.Minute), Status: "good"},
		{ID: 2, Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Status: "future"},
		{ID: 3, Start: now.Add(-200 * 24 * time.Hour), End: now.Add(-200*24*time.Hour + time.Hour), Status: "good"},
		{ID: 4, Start: now.Add(-time.Hour), End: now.Add(-2 * time.Hour), Status: "good"},
		{ID: 0, Start: now.Add(-time.Hour), End: now.Add(-30 * time.Minute), Status: "good"},
	}
	records, skipped := BuildObservationRecords(observations, now)
	if len(records) != 1 {
		t.Fatalf("expected a single usable record, got %d", len(records))
	}
	if skipped != 5 {
		t.Fatalf("expected 5 skipped entries, got %d", skipped)
	}
	if records[0].ExternalID != 1 || !records[0].ReceptionConfirmed {
		t.Fatalf("unexpected record %+v", records[0])
	}
	if records[0].EvidenceKind != EvidenceKindExternal || records[0].Applicability != ApplicabilityReception {
		t.Fatalf("records must stay labelled as external context: %+v", records[0])
	}
}

func TestBuildObservationRecordsBoundsConfidence(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	observations := []Observation{
		{ID: 10, Start: now.Add(-time.Hour), End: now.Add(-50 * time.Minute), Status: "good", VettedStatus: "good"},
		{ID: 11, Start: now.Add(-time.Hour), End: now.Add(-49 * time.Minute), Status: "unknown"},
		{ID: 12, Start: now.Add(-time.Hour), End: now.Add(-48 * time.Minute), Status: "bad", VettedStatus: "bad"},
	}
	records, _ := BuildObservationRecords(observations, now)
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	byID := map[int64]ObservationRecord{}
	for _, record := range records {
		byID[record.ExternalID] = record
		if record.Confidence > maxImportConfidence {
			t.Fatalf("record %d exceeds the external confidence ceiling: %d", record.ExternalID, record.Confidence)
		}
	}
	if byID[10].Confidence <= byID[11].Confidence || byID[11].Confidence <= byID[12].Confidence {
		t.Fatalf("confidence ordering is wrong: %+v", byID)
	}
	if byID[12].ReceptionConfirmed {
		t.Fatal("a bad observation must not be treated as a confirmed reception")
	}
}

func TestBuildObservationRecordsSanitizesFields(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	observations := []Observation{{
		ID:                     20,
		Start:                  now.Add(-time.Hour),
		End:                    now.Add(-55 * time.Minute),
		Status:                 "good",
		TransmitterDownlinkLow: downlink(-5),
		StationLatitude:        coordinate(120),
		StationLongitude:       coordinate(-3.7),
		Waterfall:              "http://insecure.example/waterfall.png",
	}}
	records, _ := BuildObservationRecords(observations, now)
	if records[0].DownlinkHz != nil {
		t.Fatal("a non-positive downlink frequency must be dropped")
	}
	if records[0].StationLatitude != nil {
		t.Fatal("an out-of-range latitude must be dropped")
	}
	if records[0].StationLongitude == nil || *records[0].StationLongitude != -3.7 {
		t.Fatal("a valid longitude must be preserved")
	}
	if records[0].WaterfallURL != "" {
		t.Fatal("non-HTTPS artefact links must be dropped")
	}
}

func TestLatestConfirmedReception(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	records := []ObservationRecord{
		{ExternalID: 1, ObservedAt: now, ReceptionConfirmed: false},
		{ExternalID: 2, ObservedAt: now.Add(-time.Hour), ReceptionConfirmed: true},
	}
	last := LatestConfirmedReception(records)
	if last == nil || !last.Equal(now.Add(-time.Hour)) {
		t.Fatalf("unexpected latest confirmed reception %v", last)
	}
	if LatestConfirmedReception(records[:1]) != nil {
		t.Fatal("expected no confirmed reception")
	}
}

func TestCheckGeometryWithoutInputsStaysUnknown(t *testing.T) {
	record := ObservationRecord{ExternalID: 1, StartedAt: time.Now().Add(-time.Hour), EndedAt: time.Now().Add(-50 * time.Minute)}
	checks := CheckGeometry([]ObservationRecord{record}, "", "")
	if checks[0].Evaluated || checks[0].Consistency != ConsistencyUnknown {
		t.Fatalf("expected an unevaluated check without orbital elements: %+v", checks[0])
	}
	record.StationLatitude = coordinate(40.4)
	checks = CheckGeometry([]ObservationRecord{record}, issLine1, "")
	if checks[0].Evaluated {
		t.Fatal("expected an unevaluated check with incomplete elements")
	}
	record.StationLongitude = coordinate(-3.7)
	checks = CheckGeometry([]ObservationRecord{record}, issLine1, issLine2)
	if !checks[0].Evaluated {
		t.Fatal("expected the check to run once station and elements are present")
	}
}

func TestCheckGeometryConfirmsReceptionUnderTheSubsatellitePoint(t *testing.T) {
	at := time.Date(2024, 7, 18, 12, 0, 0, 0, time.UTC)
	position, err := orbit.PositionAt(issLine1, issLine2, at)
	if err != nil {
		t.Fatalf("propagate: %v", err)
	}
	record := ObservationRecord{
		ExternalID:         30,
		StartedAt:          at.Add(-30 * time.Second),
		EndedAt:            at.Add(30 * time.Second),
		StationLatitude:    coordinate(position.LatitudeDeg),
		StationLongitude:   coordinate(position.LongitudeDeg),
		DownlinkHz:         downlink(437_500_000),
		ReceptionConfirmed: true,
	}
	check := CheckGeometry([]ObservationRecord{record}, issLine1, issLine2)[0]
	if !check.Visible || check.Consistency != ConsistencyConsistent {
		t.Fatalf("expected a consistent overhead pass, got %+v", check)
	}
	if check.MaxElevationDeg < 80 {
		t.Fatalf("expected a near-overhead elevation, got %v", check.MaxElevationDeg)
	}
	if check.ExpectedDopplerHz == nil {
		t.Fatal("expected a Doppler estimate when the downlink frequency is known")
	}
}

func TestCheckGeometryFlagsReceptionOutsideVisibility(t *testing.T) {
	at := time.Date(2024, 7, 18, 12, 0, 0, 0, time.UTC)
	position, err := orbit.PositionAt(issLine1, issLine2, at)
	if err != nil {
		t.Fatalf("propagate: %v", err)
	}
	antipodeLongitude := position.LongitudeDeg + 180
	if antipodeLongitude > 180 {
		antipodeLongitude -= 360
	}
	record := ObservationRecord{
		ExternalID:         31,
		StartedAt:          at.Add(-30 * time.Second),
		EndedAt:            at.Add(30 * time.Second),
		StationLatitude:    coordinate(-position.LatitudeDeg),
		StationLongitude:   coordinate(antipodeLongitude),
		ReceptionConfirmed: true,
	}
	check := CheckGeometry([]ObservationRecord{record}, issLine1, issLine2)[0]
	if check.Visible || check.Consistency != ConsistencyInconsistent {
		t.Fatalf("expected an inconsistent reception on the far side of the Earth, got %+v", check)
	}
}
