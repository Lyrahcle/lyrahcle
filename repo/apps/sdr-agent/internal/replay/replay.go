// Package replay turns a saved IQ capture back into a sequence of RF power
// readings, timestamped to match the original capture's timeline, so it can
// be re-ingested by the backend for forensic review — re-running anomaly
// detection against a historical signal without needing the original SDR
// session or hardware.
package replay

import (
	"math"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
)

// PowerWindows splits samples into consecutive, non-overlapping windows of
// windowSamples each and reduces every window to one RF power reading,
// reconstructing the capture's original timeline from meta.CapturedAt and
// meta.SampleRateHz.
//
// The power figure is a relative estimate derived from raw sample
// magnitude (10*log10(mean|s|^2)), not a calibrated absolute measurement —
// it's meaningful for spotting outliers within one replay, not for
// comparing across different hardware, gain settings, or capture sessions.
func PowerWindows(meta forensics.Metadata, samples []complex64, windowSamples int) []capture.Reading {
	if len(samples) == 0 {
		return nil
	}
	if windowSamples <= 0 || windowSamples > len(samples) {
		windowSamples = len(samples)
	}

	windowDuration := time.Second
	if meta.SampleRateHz > 0 {
		windowDuration = time.Duration(float64(windowSamples) / float64(meta.SampleRateHz) * float64(time.Second))
	}

	var out []capture.Reading
	for start, index := 0, 0; start < len(samples); start, index = start+windowSamples, index+1 {
		end := start + windowSamples
		if end > len(samples) {
			end = len(samples)
		}

		out = append(out, capture.Reading{
			FrequencyHz: int64(meta.FrequencyHz),
			PowerDbm:    meanPowerDbm(samples[start:end]),
			CapturedAt:  meta.CapturedAt.Add(time.Duration(index) * windowDuration).UTC(),
		})
	}
	return out
}

// noiseFloorDbm is reported for an all-silent window (mean power exactly
// zero) instead of -Inf, so downstream consumers never see a non-finite
// float.
const noiseFloorDbm = -140.0

func meanPowerDbm(window []complex64) float64 {
	if len(window) == 0 {
		return noiseFloorDbm
	}

	var sumSq float64
	for _, s := range window {
		re, im := float64(real(s)), float64(imag(s))
		sumSq += re*re + im*im
	}
	meanSq := sumSq / float64(len(window))
	if meanSq <= 0 {
		return noiseFloorDbm
	}
	return 10 * math.Log10(meanSq)
}
