package sigmf

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validDescriptor() Descriptor {
	return NewDescriptor("agent-1", "capture-1", 437_500_000, 240_000, time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), "RTL-SDR")
}

func TestValidateAcceptsMinimalDescriptor(t *testing.T) {
	if err := validDescriptor().Validate(); err != nil {
		t.Fatalf("expected valid descriptor, got %v", err)
	}
}

func TestValidateRejectsUnsupportedInputs(t *testing.T) {
	cases := map[string]func(d *Descriptor){
		"version":       func(d *Descriptor) { d.Global.Version = "2.0.0" },
		"datatype":      func(d *Descriptor) { d.Global.Datatype = "rf32_le" },
		"sample rate":   func(d *Descriptor) { d.Global.SampleRate = 0 },
		"nan rate":      func(d *Descriptor) { d.Global.SampleRate = math.NaN() },
		"dataset":       func(d *Descriptor) { d.Global.Dataset = "other.bin" },
		"no captures":   func(d *Descriptor) { d.Captures = nil },
		"offset start":  func(d *Descriptor) { d.Captures[0].SampleStart = 12 },
		"bad datetime":  func(d *Descriptor) { d.Captures[0].Datetime = "yesterday" },
		"bad frequency": func(d *Descriptor) { d.Captures[0].Frequency = math.Inf(1) },
	}
	for name, mutate := range cases {
		descriptor := validDescriptor()
		mutate(&descriptor)
		if err := descriptor.Validate(); err == nil {
			t.Fatalf("expected %s to be rejected", name)
		}
	}
}

func TestValidateRejectsUnorderedCaptureSegments(t *testing.T) {
	descriptor := validDescriptor()
	descriptor.Captures = append(descriptor.Captures, Capture{SampleStart: 0, Frequency: 100}, Capture{SampleStart: 5, Frequency: 100})
	if err := descriptor.Validate(); err == nil {
		t.Fatal("expected unordered capture segments to be rejected")
	}
}

func TestDecodeSamplesPerDatatype(t *testing.T) {
	cu8, err := DecodeSamples(DatatypeCU8, []byte{255, 0})
	if err != nil {
		t.Fatalf("cu8: %v", err)
	}
	if real(cu8[0]) <= 0.99 || imag(cu8[0]) >= -0.99 {
		t.Fatalf("cu8 scaling is wrong: %v", cu8[0])
	}
	ci8, err := DecodeSamples(DatatypeCI8, []byte{64, 192})
	if err != nil {
		t.Fatalf("ci8: %v", err)
	}
	if math.Abs(float64(real(ci8[0]))-0.5) > 1e-6 || math.Abs(float64(imag(ci8[0]))+0.5) > 1e-6 {
		t.Fatalf("ci8 scaling is wrong: %v", ci8[0])
	}
	ci16, err := DecodeSamples(DatatypeCI16LE, []byte{0x00, 0x40, 0x00, 0xC0})
	if err != nil {
		t.Fatalf("ci16: %v", err)
	}
	if math.Abs(float64(real(ci16[0]))-0.5) > 1e-6 || math.Abs(float64(imag(ci16[0]))+0.5) > 1e-6 {
		t.Fatalf("ci16 scaling is wrong: %v", ci16[0])
	}
}

func TestDecodeSamplesRejectsTruncatedData(t *testing.T) {
	if _, err := DecodeSamples(DatatypeCF32LE, make([]byte, 7)); err == nil {
		t.Fatal("expected truncated cf32 data to be rejected")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	samples := []complex64{complex(0.25, -0.5), complex(-1, 0.125)}
	raw, err := EncodeSamples(DatatypeCF32LE, samples)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeSamples(DatatypeCF32LE, raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i := range samples {
		if samples[i] != decoded[i] {
			t.Fatalf("sample %d changed: %v -> %v", i, samples[i], decoded[i])
		}
	}
}

func TestEncodeRejectsNonFiniteSamples(t *testing.T) {
	if _, err := EncodeSamples(DatatypeCF32LE, []complex64{complex(float32(math.Inf(1)), 0)}); err == nil {
		t.Fatal("expected non-finite sample to be rejected")
	}
}

func TestWriteAndReadRecording(t *testing.T) {
	dir := t.TempDir()
	samples := []complex64{complex(0.1, 0.2), complex(0.3, -0.4), complex(-0.5, 0.6)}
	dataPath, metaPath, err := Write(dir, "capture-1", validDescriptor(), samples)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if filepath.Ext(dataPath) != ".sigmf-data" || filepath.Ext(metaPath) != ".sigmf-meta" {
		t.Fatalf("unexpected paths %s / %s", dataPath, metaPath)
	}
	recording, err := Read(metaPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if recording.IntegrityState != "verified" {
		t.Fatalf("expected verified integrity, got %s", recording.IntegrityState)
	}
	if len(recording.Samples) != len(samples) {
		t.Fatalf("expected %d samples, got %d", len(samples), len(recording.Samples))
	}
	if got := recording.Descriptor.CenterFrequencyHz(); got != 437_500_000 {
		t.Fatalf("unexpected centre frequency %v", got)
	}
	start, ok := recording.Descriptor.StartTime()
	if !ok || !start.Equal(time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start time %v (present=%v)", start, ok)
	}
}

func TestReadDetectsTamperedData(t *testing.T) {
	dir := t.TempDir()
	dataPath, metaPath, err := Write(dir, "capture-1", validDescriptor(), []complex64{complex(0.1, 0.2), complex(0.2, 0.1)})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("read data: %v", err)
	}
	raw[0] ^= 0xFF
	if err := os.WriteFile(dataPath, raw, 0o600); err != nil {
		t.Fatalf("tamper: %v", err)
	}
	if _, err := Read(metaPath); err == nil || !strings.Contains(err.Error(), "sha512") {
		t.Fatalf("expected checksum failure, got %v", err)
	}
}

func TestReadRejectsSegmentsBeyondData(t *testing.T) {
	dir := t.TempDir()
	descriptor := validDescriptor()
	_, metaPath, err := Write(dir, "capture-1", descriptor, []complex64{complex(0.1, 0.1)})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	var parsed Descriptor
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	parsed.Captures = append(parsed.Captures, Capture{SampleStart: 4096, Frequency: 437_500_000})
	updated, err := json.Marshal(parsed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(metaPath, updated, 0o600); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	if _, err := Read(metaPath); err == nil {
		t.Fatal("expected out-of-range capture segment to be rejected")
	}
}

func TestWriteRejectsPathEscapes(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"../escape", "nested/capture", ""} {
		if _, _, err := Write(dir, name, validDescriptor(), []complex64{complex(0.1, 0.1)}); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

func TestDataPathForRequiresMetadataExtension(t *testing.T) {
	if _, err := DataPathFor("capture.json"); err == nil {
		t.Fatal("expected non-SigMF metadata path to be rejected")
	}
	path, err := DataPathFor("/tmp/capture.sigmf-meta")
	if err != nil || path != "/tmp/capture.sigmf-data" {
		t.Fatalf("unexpected data path %q (%v)", path, err)
	}
}
