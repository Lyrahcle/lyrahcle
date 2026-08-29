package forensics

import (
	"math"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/streaming"
)

func FuzzCaptureRoundTrip(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0}, "agent-1")
	f.Add([]byte{0x3f, 0x80, 0, 0, 0xbf, 0x80, 0, 0}, "../../agent")
	f.Fuzz(func(t *testing.T, raw []byte, agentID string) {
		if len(raw) < 8 || len(raw) > 4096 {
			return
		}
		sampleCount := len(raw) / 8
		if sampleCount == 0 {
			return
		}
		samples := make([]complex64, sampleCount)
		for i := range samples {
			offset := i * 8
			reBits := uint32(raw[offset])<<24 | uint32(raw[offset+1])<<16 | uint32(raw[offset+2])<<8 | uint32(raw[offset+3])
			imBits := uint32(raw[offset+4])<<24 | uint32(raw[offset+5])<<16 | uint32(raw[offset+6])<<8 | uint32(raw[offset+7])
			samples[i] = complex(math.Float32frombits(reBits), math.Float32frombits(imBits))
		}
		data := &streaming.IQData{
			Samples:    samples,
			Timestamp:  time.Unix(1_700_000_000, 0).UTC(),
			Frequency:  437_500_000,
			SampleRate: 2_000_000,
		}
		_, metaPath, err := Save(t.TempDir(), agentID, data)
		if err != nil {
			t.Fatalf("Save: %v", err)
		}
		meta, loaded, err := Load(metaPath)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if meta.SampleCount != len(samples) || len(loaded) != len(samples) {
			t.Fatalf("sample count mismatch: metadata=%d loaded=%d want=%d", meta.SampleCount, len(loaded), len(samples))
		}
	})
}
