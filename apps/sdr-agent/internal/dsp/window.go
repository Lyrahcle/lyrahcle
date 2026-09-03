package dsp

import (
	"fmt"
	"math"
	"strings"
)

type WindowKind string

const (
	WindowRectangular    WindowKind = "rectangular"
	WindowHann           WindowKind = "hann"
	WindowHamming        WindowKind = "hamming"
	WindowBlackmanHarris WindowKind = "blackman-harris"
)

type Window struct {
	Kind           WindowKind
	Taps           []float64
	CoherentGain   float64
	NoisePowerGain float64
}

func ParseWindowKind(value string) (WindowKind, error) {
	switch WindowKind(strings.ToLower(strings.TrimSpace(value))) {
	case "", WindowHann:
		return WindowHann, nil
	case WindowRectangular:
		return WindowRectangular, nil
	case WindowHamming:
		return WindowHamming, nil
	case WindowBlackmanHarris:
		return WindowBlackmanHarris, nil
	default:
		return "", fmt.Errorf("dsp: unsupported window %q", value)
	}
}

func NewWindow(kind WindowKind, size int) (Window, error) {
	if size <= 0 {
		return Window{}, fmt.Errorf("dsp: window size must be positive, got %d", size)
	}
	taps := make([]float64, size)
	denominator := float64(size - 1)
	if denominator == 0 {
		denominator = 1
	}
	switch kind {
	case WindowRectangular:
		for i := range taps {
			taps[i] = 1
		}
	case WindowHann:
		for i := range taps {
			taps[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/denominator)
		}
	case WindowHamming:
		for i := range taps {
			taps[i] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/denominator)
		}
	case WindowBlackmanHarris:
		const a0, a1, a2, a3 = 0.35875, 0.48829, 0.14128, 0.01168
		for i := range taps {
			x := 2 * math.Pi * float64(i) / denominator
			taps[i] = a0 - a1*math.Cos(x) + a2*math.Cos(2*x) - a3*math.Cos(3*x)
		}
	default:
		return Window{}, fmt.Errorf("dsp: unsupported window %q", kind)
	}

	var sum, sumSquares float64
	for _, tap := range taps {
		sum += tap
		sumSquares += tap * tap
	}
	n := float64(size)
	return Window{
		Kind:           kind,
		Taps:           taps,
		CoherentGain:   sum / n,
		NoisePowerGain: sumSquares / n,
	}, nil
}

func (w Window) Apply(samples []complex128) []complex128 {
	out := make([]complex128, len(samples))
	for i, s := range samples {
		if i >= len(w.Taps) {
			break
		}
		tap := complex(w.Taps[i], 0)
		out[i] = s * tap
	}
	return out
}

func (w Window) EquivalentNoiseBandwidthBins() float64 {
	if w.CoherentGain == 0 {
		return 1
	}
	return w.NoisePowerGain / (w.CoherentGain * w.CoherentGain)
}
