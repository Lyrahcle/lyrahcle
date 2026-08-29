package replay

import (
	"math"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
)

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

const noiseFloorDbm = -140.0

func meanPowerDbm(window []complex64) float64 {
	if len(window) == 0 {
		return noiseFloorDbm
	}

	var sumSq float64
	valid := 0
	for _, s := range window {
		re, im := float64(real(s)), float64(imag(s))
		power := re*re + im*im
		if math.IsNaN(power) || math.IsInf(power, 0) {
			continue
		}
		sumSq += power
		valid++
	}
	if valid == 0 {
		return noiseFloorDbm
	}
	meanSq := sumSq / float64(valid)
	if meanSq <= 0 || math.IsNaN(meanSq) || math.IsInf(meanSq, 0) {
		return noiseFloorDbm
	}
	return 10 * math.Log10(meanSq)
}
