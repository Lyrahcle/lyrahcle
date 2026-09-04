package replay

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/sigmf"
)

func writeUnsignedByteRecording(t *testing.T, dir, name string, sampleRate, centerFrequency float64, offsetHz float64, sampleCount int) string {
	t.Helper()
	raw := make([]byte, 0, sampleCount*2)
	for n := 0; n < sampleCount; n++ {
		phase := 2 * math.Pi * offsetHz * float64(n) / sampleRate
		i := math.Round(math.Cos(phase)*100 + 127.5)
		q := math.Round(math.Sin(phase)*100 + 127.5)
		raw = append(raw, byte(clampByte(i)), byte(clampByte(q)))
	}
	dataPath := filepath.Join(dir, name+sigmf.DataExtension)
	if err := os.WriteFile(dataPath, raw, 0o600); err != nil {
		t.Fatalf("write data: %v", err)
	}
	descriptor := sigmf.Descriptor{
		Global: sigmf.Global{
			Datatype:   sigmf.DatatypeCU8,
			SampleRate: sampleRate,
			Version:    sigmf.Version,
			Hardware:   "public community recording",
		},
		Captures: []sigmf.Capture{{
			SampleStart: 0,
			Frequency:   centerFrequency,
			Datetime:    time.Date(2026, 4, 2, 9, 30, 0, 0, time.UTC).Format(time.RFC3339),
		}},
	}
	metaBytes, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	metaPath := filepath.Join(dir, name+sigmf.MetaExtension)
	if err := os.WriteFile(metaPath, metaBytes, 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	return metaPath
}

func clampByte(value float64) int {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return int(value)
}

func TestSpectralWindowsFromUnsignedByteSigMFRecording(t *testing.T) {
	dir := t.TempDir()
	const sampleRate = 240_000.0
	const centerFrequency = 437_500_000.0
	const offsetHz = 12_000.0
	metaPath := writeUnsignedByteRecording(t, dir, "community", sampleRate, centerFrequency, offsetHz, 65536)

	meta, samples, err := forensics.LoadAnyWithin(dir, filepath.Base(metaPath))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if meta.SampleRateHz != 240_000 || meta.FrequencyHz != 437_500_000 {
		t.Fatalf("unexpected tuning %d Hz at %d Sps", meta.FrequencyHz, meta.SampleRateHz)
	}
	if meta.Extra["evidence_kind"] != "external_recording" {
		t.Fatalf("expected the recording to be labelled as external, got %v", meta.Extra)
	}

	readings, err := SpectralWindows(meta, samples, SpectralOptions{FFTSize: 4096, Window: "hann", OverlapRatio: 0.5, WindowSamples: 32768})
	if err != nil {
		t.Fatalf("spectral: %v", err)
	}
	if len(readings) == 0 || readings[0].Spectral == nil {
		t.Fatal("expected spectral readings")
	}
	if math.Abs(readings[0].Spectral.CarrierOffsetHz-offsetHz) > 50 {
		t.Fatalf("carrier offset %v Hz is not within 50 Hz of the injected %v Hz", readings[0].Spectral.CarrierOffsetHz, offsetHz)
	}
	if readings[0].Spectral.SNRDb <= 10 {
		t.Fatalf("expected a strong tone, got %v dB SNR", readings[0].Spectral.SNRDb)
	}
}

func TestDopplerTrackFromSigMFRecording(t *testing.T) {
	dir := t.TempDir()
	const sampleRate = 240_000.0
	const centerFrequency = 437_500_000.0
	metaPath := writeUnsignedByteRecording(t, dir, "doppler", sampleRate, centerFrequency, 9_000, 131072)

	meta, samples, err := forensics.LoadAnyWithin(dir, filepath.Base(metaPath))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	readings, err := SpectralWindows(meta, samples, SpectralOptions{FFTSize: 4096, Window: "hann", OverlapRatio: 0.5, WindowSamples: 32768})
	if err != nil {
		t.Fatalf("spectral: %v", err)
	}
	track, err := DopplerTrackFromReadings(readings, centerFrequency, 5)
	if err != nil {
		t.Fatalf("doppler track: %v", err)
	}
	if len(track) < 2 {
		t.Fatalf("expected several track points, got %d", len(track))
	}
	if math.Abs(track[0].OffsetHz-9_000) > 50 {
		t.Fatalf("unexpected offset %v Hz", track[0].OffsetHz)
	}
}
