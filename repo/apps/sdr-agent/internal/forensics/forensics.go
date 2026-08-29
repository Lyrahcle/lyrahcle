// Package forensics persists raw IQ captures to disk and reads them back,
// so a capture taken today can be replayed and re-analyzed later — by this
// agent, a different one, or offline — without needing the original SDR
// session or hardware.
package forensics

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/streaming"
)

// dataFormat identifies the raw sample encoding: interleaved little-endian
// float32 I/Q pairs, no header. Compatible with GNU Radio's "gr_complex"
// / .cf32 convention, so captures can be opened in other SDR tooling too.
const dataFormat = "cf32"

// Metadata describes a saved IQ capture: everything needed to replay it
// without the original device or session.
type Metadata struct {
	ID           string            `json:"id"`
	AgentID      string            `json:"agent_id"`
	FrequencyHz  uint64            `json:"frequency_hz"`
	SampleRateHz uint64            `json:"sample_rate_hz"`
	CapturedAt   time.Time         `json:"captured_at"`
	SampleCount  int               `json:"sample_count"`
	Format       string            `json:"format"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// Save writes an IQ capture to dir as two files sharing a basename: the
// raw samples (<id>.cf32) and their metadata (<id>.json). Returns both
// paths.
func Save(dir, agentID string, data *streaming.IQData) (dataPath, metaPath string, err error) {
	if data == nil || len(data.Samples) == 0 {
		return "", "", fmt.Errorf("forensics: capture has no samples")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("forensics: create dir: %w", err)
	}

	id := fmt.Sprintf("%s_%d", sanitize(agentID), data.Timestamp.UnixNano())
	dataPath = filepath.Join(dir, id+"."+dataFormat)
	metaPath = filepath.Join(dir, id+".json")

	buf := make([]byte, 0, len(data.Samples)*8)
	for _, s := range data.Samples {
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(real(s)))
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(imag(s)))
	}
	if err := os.WriteFile(dataPath, buf, 0o644); err != nil {
		return "", "", fmt.Errorf("forensics: write samples: %w", err)
	}

	meta := Metadata{
		ID:           id,
		AgentID:      agentID,
		FrequencyHz:  data.Frequency,
		SampleRateHz: data.SampleRate,
		CapturedAt:   data.Timestamp.UTC(),
		SampleCount:  len(data.Samples),
		Format:       dataFormat,
		Extra:        data.Metadata,
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		os.Remove(dataPath)
		return "", "", fmt.Errorf("forensics: marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		os.Remove(dataPath)
		return "", "", fmt.Errorf("forensics: write metadata: %w", err)
	}

	return dataPath, metaPath, nil
}

// Load reads back a capture previously written by Save, given the path to
// its .json metadata file.
func Load(metaPath string) (Metadata, []complex64, error) {
	meta, err := readMetadata(metaPath)
	if err != nil {
		return Metadata{}, nil, err
	}
	if meta.Format != dataFormat {
		return Metadata{}, nil, fmt.Errorf("forensics: unsupported capture format %q", meta.Format)
	}

	dataPath := strings.TrimSuffix(metaPath, ".json") + "." + dataFormat
	raw, err := os.ReadFile(dataPath)
	if err != nil {
		return Metadata{}, nil, fmt.Errorf("forensics: read samples: %w", err)
	}
	if len(raw)%8 != 0 {
		return Metadata{}, nil, fmt.Errorf("forensics: sample file size %d is not a multiple of 8 bytes", len(raw))
	}

	samples := make([]complex64, len(raw)/8)
	for i := range samples {
		off := i * 8
		re := math.Float32frombits(binary.LittleEndian.Uint32(raw[off:]))
		im := math.Float32frombits(binary.LittleEndian.Uint32(raw[off+4:]))
		samples[i] = complex(re, im)
	}
	if meta.SampleCount != 0 && meta.SampleCount != len(samples) {
		return meta, samples, fmt.Errorf(
			"forensics: metadata declares %d samples, file has %d — capture may be truncated or corrupt",
			meta.SampleCount, len(samples),
		)
	}

	return meta, samples, nil
}

// List returns metadata for every capture found in dir, most recent first.
// A missing directory is treated as an empty list, not an error.
func List(dir string) ([]Metadata, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("forensics: list dir: %w", err)
	}

	var out []Metadata
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		meta, err := readMetadata(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CapturedAt.After(out[j].CapturedAt) })
	return out, nil
}

func readMetadata(metaPath string) (Metadata, error) {
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return Metadata{}, fmt.Errorf("forensics: read metadata: %w", err)
	}
	var meta Metadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return Metadata{}, fmt.Errorf("forensics: parse metadata: %w", err)
	}
	return meta, nil
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "agent"
	}
	return b.String()
}
