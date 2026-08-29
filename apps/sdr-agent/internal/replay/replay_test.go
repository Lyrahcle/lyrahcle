package replay

import (
	"math"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
)

func TestPowerWindowsCountAndTiming(t *testing.T) {
	start := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	meta := forensics.Metadata{
		FrequencyHz:  437500000,
		SampleRateHz: 1000,
		CapturedAt:   start,
	}
	samples := make([]complex64, 350)

	readings := PowerWindows(meta, samples, 100)

	if len(readings) != 4 {
		t.Fatalf("got %d readings, want 4", len(readings))
	}
	for i, r := range readings {
		if r.FrequencyHz != 437500000 {
			t.Errorf("reading %d FrequencyHz = %d, want 437500000", i, r.FrequencyHz)
		}
		wantTS := start.Add(time.Duration(i) * 100 * time.Millisecond)
		if !r.CapturedAt.Equal(wantTS) {
			t.Errorf("reading %d CapturedAt = %v, want %v", i, r.CapturedAt, wantTS)
		}
	}
}

func TestPowerWindowsSilenceIsNoiseFloor(t *testing.T) {
	meta := forensics.Metadata{SampleRateHz: 2_000_000, CapturedAt: time.Now()}
	samples := make([]complex64, 10)

	readings := PowerWindows(meta, samples, 10)

	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1", len(readings))
	}
	if readings[0].PowerDbm != noiseFloorDbm {
		t.Errorf("PowerDbm = %v, want noise floor %v", readings[0].PowerDbm, noiseFloorDbm)
	}
}

func TestPowerWindowsKnownMagnitude(t *testing.T) {

	meta := forensics.Metadata{SampleRateHz: 1000, CapturedAt: time.Now()}
	samples := make([]complex64, 8)
	for i := range samples {
		samples[i] = complex64(complex(3, 4))
	}

	readings := PowerWindows(meta, samples, 8)

	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1", len(readings))
	}
	want := 10 * math.Log10(25)
	if math.Abs(readings[0].PowerDbm-want) > 0.01 {
		t.Errorf("PowerDbm = %.4f, want %.4f", readings[0].PowerDbm, want)
	}
}

func TestPowerWindowsEmptySamples(t *testing.T) {
	meta := forensics.Metadata{SampleRateHz: 1000, CapturedAt: time.Now()}
	if got := PowerWindows(meta, nil, 100); got != nil {
		t.Errorf("got %d readings for empty input, want nil", len(got))
	}
}

func TestPowerWindowsOversizedWindowClampsToOneWindow(t *testing.T) {
	meta := forensics.Metadata{SampleRateHz: 1000, CapturedAt: time.Now()}
	samples := make([]complex64, 10)

	readings := PowerWindows(meta, samples, 10_000)

	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1", len(readings))
	}
}
