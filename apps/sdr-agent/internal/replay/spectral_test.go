package replay

import (
	"math"
	"math/cmplx"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
)

func chirpCapture(count int, sampleRate, startOffsetHz, endOffsetHz float64) []complex64 {
	out := make([]complex64, count)
	phase := 0.0
	for i := range out {
		fraction := float64(i) / float64(count-1)
		offset := startOffsetHz + (endOffsetHz-startOffsetHz)*fraction
		phase += 2 * math.Pi * offset / sampleRate
		value := cmplx.Exp(complex(0, phase))
		out[i] = complex(float32(real(value)), float32(imag(value)))
	}
	return out
}

func steadyTone(count int, sampleRate, offsetHz float64) []complex64 {
	return chirpCapture(count, sampleRate, offsetHz, offsetHz)
}

func testMetadata(sampleRate, frequency uint64, count int) forensics.Metadata {
	return forensics.Metadata{
		ID:           "test",
		AgentID:      "agent-1",
		FrequencyHz:  frequency,
		SampleRateHz: sampleRate,
		CapturedAt:   time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC),
		SampleCount:  count,
		Format:       "cf32",
	}
}

func TestSpectralWindowsExtractsCarrier(t *testing.T) {
	const sampleRate = 2_048_000
	const frequency = 437_500_000
	samples := steadyTone(131072, sampleRate, 40_000)
	meta := testMetadata(sampleRate, frequency, len(samples))

	readings, err := SpectralWindows(meta, samples, SpectralOptions{FFTSize: 4096, WindowSamples: 32768, Window: "hann"})
	if err != nil {
		t.Fatal(err)
	}
	if len(readings) != 4 {
		t.Fatalf("got %d readings, want 4", len(readings))
	}
	for i, reading := range readings {
		if reading.Spectral == nil {
			t.Fatalf("reading %d has no spectral metrics", i)
		}
		if math.Abs(reading.Spectral.CarrierOffsetHz-40_000) > 100 {
			t.Fatalf("reading %d: offset %.1f Hz, want 40000 Hz", i, reading.Spectral.CarrierOffsetHz)
		}
		if reading.FrequencyHz != frequency {
			t.Fatalf("reading %d: frequency %d, want %d", i, reading.FrequencyHz, frequency)
		}
	}
	if !readings[1].CapturedAt.After(readings[0].CapturedAt) {
		t.Fatal("window timestamps must advance")
	}
}

func TestSpectralWindowsTracksAChirp(t *testing.T) {
	const sampleRate = 2_048_000
	samples := chirpCapture(131072, sampleRate, -60_000, 60_000)
	meta := testMetadata(sampleRate, 437_500_000, len(samples))

	readings, err := SpectralWindows(meta, samples, SpectralOptions{FFTSize: 4096, WindowSamples: 32768})
	if err != nil {
		t.Fatal(err)
	}
	if readings[0].Spectral.CarrierOffsetHz >= readings[len(readings)-1].Spectral.CarrierOffsetHz {
		t.Fatalf("a rising chirp must produce increasing offsets: first=%.1f last=%.1f",
			readings[0].Spectral.CarrierOffsetHz, readings[len(readings)-1].Spectral.CarrierOffsetHz)
	}
	for i := 1; i < len(readings); i++ {
		if readings[i].Spectral.CarrierOffsetHz <= readings[i-1].Spectral.CarrierOffsetHz {
			t.Fatalf("window %d offset %.1f did not increase over %.1f", i, readings[i].Spectral.CarrierOffsetHz, readings[i-1].Spectral.CarrierOffsetHz)
		}
	}
}

func TestSpectralWindowsValidatesInput(t *testing.T) {
	const sampleRate = 1_024_000
	samples := steadyTone(8192, sampleRate, 1000)
	meta := testMetadata(sampleRate, 100_000_000, len(samples))

	if _, err := SpectralWindows(meta, nil, SpectralOptions{}); err == nil {
		t.Fatal("expected an error for an empty capture")
	}
	if _, err := SpectralWindows(forensics.Metadata{}, samples, SpectralOptions{}); err == nil {
		t.Fatal("expected an error when the sample rate is missing")
	}
	if _, err := SpectralWindows(meta, samples, SpectralOptions{FFTSize: 3000}); err == nil {
		t.Fatal("expected an error for a non power-of-two FFT size")
	}
	if _, err := SpectralWindows(meta, samples, SpectralOptions{FFTSize: 16384}); err == nil {
		t.Fatal("expected an error when the capture is shorter than the FFT size")
	}
	if _, err := SpectralWindows(meta, samples, SpectralOptions{FFTSize: 1024, Window: "kaiser"}); err == nil {
		t.Fatal("expected an error for an unsupported window")
	}
}

func TestDopplerTrackConvertsToRangeRate(t *testing.T) {
	const nominal = 437_500_000.0
	readings := []capture.Reading{
		{CapturedAt: time.Now(), Spectral: &capture.SpectralMetrics{CarrierFrequencyHz: nominal + 9_000, SNRDb: 25, EstimatorMethod: "psd-peak+kay"}},
		{CapturedAt: time.Now(), Spectral: &capture.SpectralMetrics{CarrierFrequencyHz: nominal, SNRDb: 24, EstimatorMethod: "psd-peak+kay"}},
		{CapturedAt: time.Now(), Spectral: &capture.SpectralMetrics{CarrierFrequencyHz: nominal - 9_000, SNRDb: 23, EstimatorMethod: "psd-peak+kay"}},
		{CapturedAt: time.Now(), Spectral: &capture.SpectralMetrics{CarrierFrequencyHz: nominal - 100, SNRDb: 2, EstimatorMethod: "psd-peak"}},
	}

	track, err := DopplerTrackFromReadings(readings, nominal, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(track) != 3 {
		t.Fatalf("got %d points, want 3 after the SNR gate", len(track))
	}
	if track[0].RangeRateMps >= 0 {
		t.Fatalf("a positive Doppler offset means the satellite is approaching, want a negative range rate, got %.1f", track[0].RangeRateMps)
	}
	if track[2].RangeRateMps <= 0 {
		t.Fatalf("a negative Doppler offset means the satellite is receding, want a positive range rate, got %.1f", track[2].RangeRateMps)
	}
	if math.Abs(track[1].RangeRateMps) > 1e-6 {
		t.Fatalf("a zero offset must give a zero range rate, got %.6f", track[1].RangeRateMps)
	}
}

func TestDopplerTrackRejectsEmptyResult(t *testing.T) {
	readings := []capture.Reading{
		{Spectral: &capture.SpectralMetrics{CarrierFrequencyHz: 437_500_000, SNRDb: 1}},
	}
	if _, err := DopplerTrackFromReadings(readings, 437_500_000, 10); err == nil {
		t.Fatal("expected an error when no window passes the SNR gate")
	}
	if _, err := DopplerTrackFromReadings(readings, 0, 0); err == nil {
		t.Fatal("expected an error for a zero nominal frequency")
	}
}
