package forensics

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/sigmf"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/streaming"
)

func TestSaveSigMFRoundTrip(t *testing.T) {
	dir := t.TempDir()
	data := &streaming.IQData{
		Samples:    []complex64{complex(0.1, 0.2), complex(0.3, 0.4)},
		Timestamp:  time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
		Frequency:  145_800_000,
		SampleRate: 250_000,
		Metadata:   map[string]string{"device": "rtltcp"},
	}
	dataPath, metaPath, err := SaveSigMF(dir, "station-01", data)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if filepath.Ext(dataPath) != ".sigmf-data" {
		t.Fatalf("unexpected data path %s", dataPath)
	}

	meta, samples, err := LoadAnyWithin(dir, filepath.Base(metaPath))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if meta.FrequencyHz != 145_800_000 || meta.SampleRateHz != 250_000 {
		t.Fatalf("unexpected tuning: %d Hz at %d Sps", meta.FrequencyHz, meta.SampleRateHz)
	}
	if !meta.CapturedAt.Equal(data.Timestamp) {
		t.Fatalf("unexpected capture time %v", meta.CapturedAt)
	}
	if meta.AgentID != "station-01" {
		t.Fatalf("unexpected agent %q", meta.AgentID)
	}
	if meta.Extra["source"] != "sigmf" || meta.Extra["integrity"] != "verified" {
		t.Fatalf("unexpected provenance %v", meta.Extra)
	}
	if meta.Extra["evidence_kind"] != "agent_measured" || meta.Extra["timestamp_from"] != "metadata" {
		t.Fatalf("unexpected evidence labelling %v", meta.Extra)
	}
	if len(samples) != len(data.Samples) {
		t.Fatalf("expected %d samples, got %d", len(data.Samples), len(samples))
	}
}

func TestLoadSigMFMarksThirdPartyRecordings(t *testing.T) {
	dir := t.TempDir()
	descriptor := sigmf.Descriptor{
		Global: sigmf.Global{
			Datatype:   sigmf.DatatypeCF32LE,
			SampleRate: 100_000,
			Version:    sigmf.Version,
			Author:     "community observer",
		},
		Captures: []sigmf.Capture{{SampleStart: 0, Frequency: 437_000_000}},
	}
	_, metaPath, err := sigmf.Write(dir, "external", descriptor, []complex64{complex(0.2, 0.1), complex(0.1, 0.2)})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	meta, _, err := LoadAnyWithin(dir, filepath.Base(metaPath))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if meta.Extra["evidence_kind"] != "external_recording" {
		t.Fatalf("expected external recordings to be labelled, got %v", meta.Extra)
	}
	if meta.Extra["timestamp_from"] != "file_modification_time" {
		t.Fatalf("expected fallback timestamp provenance, got %v", meta.Extra)
	}
	if meta.AgentID != "sigmf-import" {
		t.Fatalf("unexpected agent %q", meta.AgentID)
	}
}

func TestLoadSigMFRejectsPathEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	descriptor := sigmf.NewDescriptor("agent", "capture", 100_000_000, 100_000, time.Now().UTC(), "")
	if _, _, err := sigmf.Write(outside, "external", descriptor, []complex64{complex(0.1, 0.1)}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := LoadAnyWithin(dir, "../"+filepath.Base(outside)+"/external.sigmf-meta"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestLoadSigMFRequiresDataFile(t *testing.T) {
	dir := t.TempDir()
	descriptor := sigmf.NewDescriptor("agent", "capture", 100_000_000, 100_000, time.Now().UTC(), "")
	dataPath, metaPath, err := sigmf.Write(dir, "capture", descriptor, []complex64{complex(0.1, 0.1)})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Remove(dataPath); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, _, err := LoadAnyWithin(dir, filepath.Base(metaPath)); err == nil {
		t.Fatal("expected missing data file to be rejected")
	}
}

func TestLoadAnyWithinStillReadsNativeCaptures(t *testing.T) {
	dir := t.TempDir()
	data := &streaming.IQData{
		Samples:    []complex64{complex(0.1, 0.2)},
		Timestamp:  time.Now().UTC(),
		Frequency:  400_000_000,
		SampleRate: 200_000,
	}
	_, metaPath, err := Save(dir, "agent", data)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, samples, err := LoadAnyWithin(dir, filepath.Base(metaPath)); err != nil || len(samples) != 1 {
		t.Fatalf("expected native capture to load, got %d samples (%v)", len(samples), err)
	}
}
