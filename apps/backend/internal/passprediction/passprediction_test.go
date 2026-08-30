package passprediction

import (
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/orbitalcontext"
)

const (
	issLine1 = "1 25544U 98067A   24200.50000000  .00016717  00000+0  30182-3 0  9991"
	issLine2 = "2 25544  51.6400 100.0000 0006000  60.0000  40.0000 15.50000000123456"
)

func TestPredictValidatesRequest(t *testing.T) {
	_, err := Predict(issLine1, issLine2, Request{
		Station: orbitalcontext.Station{},
		From:    time.Now(),
		Horizon: 0,
		Limit:   1,
	})
	if err == nil {
		t.Fatal("expected horizon validation error")
	}
}

func TestPredictReturnsBoundedChronologicalPasses(t *testing.T) {
	from := time.Date(2024, 7, 18, 0, 0, 0, 0, time.UTC)
	result, err := Predict(issLine1, issLine2, Request{
		Station:         orbitalcontext.Station{LatitudeDeg: 40.4168, LongitudeDeg: -3.7038, AltitudeM: 667},
		From:            from,
		Horizon:         24 * time.Hour,
		MinElevationDeg: 0,
		Limit:           10,
	})
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if result.From != from || result.Until != from.Add(24*time.Hour) {
		t.Fatalf("unexpected prediction window: %+v", result)
	}
	if len(result.Passes) == 0 {
		t.Fatal("expected at least one pass in a 24-hour ISS window")
	}
	if len(result.Passes) > 10 {
		t.Fatalf("expected limit to be honored, got %d", len(result.Passes))
	}
	for _, pass := range result.Passes {
		if pass.LOS.Before(pass.AOS) {
			t.Fatalf("LOS before AOS: %+v", pass)
		}
		if pass.PeakAt.Before(pass.AOS) || pass.PeakAt.After(pass.LOS) {
			t.Fatalf("peak outside pass: %+v", pass)
		}
		if pass.DurationSeconds < 0 {
			t.Fatalf("negative duration: %+v", pass)
		}
	}
}
