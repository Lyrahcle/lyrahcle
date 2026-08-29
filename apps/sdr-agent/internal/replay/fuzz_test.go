package replay

import (
	"math"
	"testing"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
)

func FuzzPowerWindows(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0}, uint16(1))
	f.Add([]byte{0x3f, 0x80, 0, 0, 0xbf, 0x80, 0, 0}, uint16(64))
	f.Fuzz(func(t *testing.T, raw []byte, requestedWindow uint16) {
		if len(raw) > 8192 {
			return
		}
		sampleCount := len(raw) / 8
		samples := make([]complex64, sampleCount)
		for i := range samples {
			offset := i * 8
			reBits := uint32(raw[offset])<<24 | uint32(raw[offset+1])<<16 | uint32(raw[offset+2])<<8 | uint32(raw[offset+3])
			imBits := uint32(raw[offset+4])<<24 | uint32(raw[offset+5])<<16 | uint32(raw[offset+6])<<8 | uint32(raw[offset+7])
			samples[i] = complex(math.Float32frombits(reBits), math.Float32frombits(imBits))
		}
		window := int(requestedWindow)
		meta := forensics.Metadata{
			FrequencyHz:  437_500_000,
			SampleRateHz: 2_000_000,
			CapturedAt:   time.Unix(1_700_000_000, 0).UTC(),
		}
		readings := PowerWindows(meta, samples, window)
		if len(samples) == 0 && len(readings) != 0 {
			t.Fatalf("empty input produced %d readings", len(readings))
		}
		for _, reading := range readings {
			if math.IsNaN(reading.PowerDbm) || math.IsInf(reading.PowerDbm, 0) {
				t.Fatalf("invalid power value %v", reading.PowerDbm)
			}
		}
	})
}
