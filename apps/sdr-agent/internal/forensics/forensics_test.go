package forensics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/streaming"
)

func sampleData(ts time.Time) *streaming.IQData {
	return &streaming.IQData{
		Samples:    []complex64{1 + 2i, -3 + 4i, 0.5 - 0.5i},
		Timestamp:  ts,
		Frequency:  437500000,
		SampleRate: 2_000_000,
		Metadata:   map[string]string{"gain": "30.0", "antenna": "RX"},
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

	dataPath, metaPath, err := Save(dir, "sdr-agent-madrid", sampleData(ts))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if filepath.Dir(dataPath) != dir || filepath.Dir(metaPath) != dir {
		t.Fatalf("expected files under %s, got %s and %s", dir, dataPath, metaPath)
	}

	meta, samples, err := Load(metaPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if meta.AgentID != "sdr-agent-madrid" {
		t.Errorf("AgentID = %q, want sdr-agent-madrid", meta.AgentID)
	}
	if meta.FrequencyHz != 437500000 {
		t.Errorf("FrequencyHz = %d, want 437500000", meta.FrequencyHz)
	}
	if meta.SampleRateHz != 2_000_000 {
		t.Errorf("SampleRateHz = %d, want 2000000", meta.SampleRateHz)
	}
	if !meta.CapturedAt.Equal(ts) {
		t.Errorf("CapturedAt = %v, want %v", meta.CapturedAt, ts)
	}
	if meta.SampleCount != 3 {
		t.Errorf("SampleCount = %d, want 3", meta.SampleCount)
	}
	if meta.Extra["antenna"] != "RX" {
		t.Errorf("Extra[antenna] = %q, want RX", meta.Extra["antenna"])
	}

	want := sampleData(ts).Samples
	if len(samples) != len(want) {
		t.Fatalf("got %d samples, want %d", len(samples), len(want))
	}
	for i := range want {
		if samples[i] != want[i] {
			t.Errorf("sample %d = %v, want %v", i, samples[i], want[i])
		}
	}
}

func TestSaveRejectsEmptyCapture(t *testing.T) {
	dir := t.TempDir()
	_, _, err := Save(dir, "agent", &streaming.IQData{})
	if err == nil {
		t.Fatal("expected an error for an empty capture, got nil")
	}
}

func TestListMostRecentFirst(t *testing.T) {
	dir := t.TempDir()
	older := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)

	if _, _, err := Save(dir, "agent-a", sampleData(older)); err != nil {
		t.Fatalf("Save older: %v", err)
	}
	if _, _, err := Save(dir, "agent-b", sampleData(newer)); err != nil {
		t.Fatalf("Save newer: %v", err)
	}

	list, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d captures, want 2", len(list))
	}
	if list[0].AgentID != "agent-b" || list[1].AgentID != "agent-a" {
		t.Errorf("List order = [%s, %s], want [agent-b, agent-a] (most recent first)", list[0].AgentID, list[1].AgentID)
	}
}

func TestListOnMissingDirReturnsEmpty(t *testing.T) {
	list, err := List(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d captures, want 0", len(list))
	}
}

func TestLoadRejectsSampleCountMismatch(t *testing.T) {
	dir := t.TempDir()
	_, metaPath, err := Save(dir, "agent", sampleData(time.Now()))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	meta, err := readMetadata(metaPath)
	if err != nil {
		t.Fatalf("readMetadata: %v", err)
	}
	meta.SampleCount = 999
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		t.Fatalf("rewrite metadata: %v", err)
	}

	if _, _, err := Load(metaPath); err == nil {
		t.Fatal("expected a sample-count mismatch error, got nil")
	}
}

func TestSaveUsesPrivatePermissionsAndBoundsFilename(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "captures")
	agentID := strings.Repeat("agent/../", 100)
	dataPath, metaPath, err := Save(dir, agentID, sampleData(time.Now()))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir permissions = %o, want 700", dirInfo.Mode().Perm())
	}

	for _, path := range []string{dataPath, metaPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s permissions = %o, want 600", path, info.Mode().Perm())
		}
		if len(filepath.Base(path)) > 128 {
			t.Fatalf("filename length = %d, want <= 128", len(filepath.Base(path)))
		}
	}
}
