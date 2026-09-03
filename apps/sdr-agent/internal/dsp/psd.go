package dsp

import (
	"fmt"
	"math"
	"sort"
)

type PSDOptions struct {
	FFTSize      int
	OverlapRatio float64
	Window       WindowKind
	SampleRateHz float64
	CenterFreqHz float64
}

type PSD struct {
	FFTSize      int
	Segments     int
	SampleRateHz float64
	CenterFreqHz float64
	BinWidthHz   float64
	PowerDbFS    []float64
	FrequencyHz  []float64
	windowENBW   float64
}

func (o PSDOptions) normalized() (PSDOptions, error) {
	out := o
	if out.FFTSize == 0 {
		out.FFTSize = 4096
	}
	if !IsPowerOfTwo(out.FFTSize) {
		return out, fmt.Errorf("dsp: FFT size %d is not a power of two", out.FFTSize)
	}
	if out.OverlapRatio < 0 || out.OverlapRatio >= 1 {
		return out, fmt.Errorf("dsp: overlap ratio must be in [0,1), got %v", out.OverlapRatio)
	}
	if out.OverlapRatio == 0 {
		out.OverlapRatio = 0.5
	}
	if out.SampleRateHz <= 0 {
		return out, fmt.Errorf("dsp: sample rate must be positive, got %v", out.SampleRateHz)
	}
	kind, err := ParseWindowKind(string(out.Window))
	if err != nil {
		return out, err
	}
	out.Window = kind
	return out, nil
}

func Welch(samples []complex128, opts PSDOptions) (*PSD, error) {
	opts, err := opts.normalized()
	if err != nil {
		return nil, err
	}
	if len(samples) < opts.FFTSize {
		return nil, fmt.Errorf("dsp: need at least %d samples for a %d-point PSD, got %d", opts.FFTSize, opts.FFTSize, len(samples))
	}

	window, err := NewWindow(opts.Window, opts.FFTSize)
	if err != nil {
		return nil, err
	}

	step := int(float64(opts.FFTSize) * (1 - opts.OverlapRatio))
	if step < 1 {
		step = 1
	}

	accumulator := make([]float64, opts.FFTSize)
	segments := 0
	normalization := float64(opts.FFTSize) * float64(opts.FFTSize) * window.NoisePowerGain

	for start := 0; start+opts.FFTSize <= len(samples); start += step {
		segment := window.Apply(samples[start : start+opts.FFTSize])
		spectrum, err := FFT(segment)
		if err != nil {
			return nil, err
		}
		for i, bin := range spectrum {
			re, im := real(bin), imag(bin)
			accumulator[i] += (re*re + im*im) / normalization
		}
		segments++
	}
	if segments == 0 {
		return nil, fmt.Errorf("dsp: no complete FFT segment in %d samples", len(samples))
	}

	powerDbFS := make([]float64, opts.FFTSize)
	for i, total := range accumulator {
		mean := total / float64(segments)
		powerDbFS[i] = toDb(mean)
	}
	powerDbFS = ShiftFloat(powerDbFS)

	binWidth := opts.SampleRateHz / float64(opts.FFTSize)
	frequencies := make([]float64, opts.FFTSize)
	for i := range frequencies {
		frequencies[i] = opts.CenterFreqHz + (float64(i)-float64(opts.FFTSize)/2)*binWidth
	}

	return &PSD{
		FFTSize:      opts.FFTSize,
		Segments:     segments,
		SampleRateHz: opts.SampleRateHz,
		CenterFreqHz: opts.CenterFreqHz,
		BinWidthHz:   binWidth,
		PowerDbFS:    powerDbFS,
		FrequencyHz:  frequencies,
		windowENBW:   window.EquivalentNoiseBandwidthBins(),
	}, nil
}

func toDb(power float64) float64 {
	if power <= 0 {
		return -300
	}
	return 10 * math.Log10(power)
}

func (p *PSD) NoiseFloorDbFS() float64 {
	if len(p.PowerDbFS) == 0 {
		return -300
	}
	sorted := append([]float64(nil), p.PowerDbFS...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

type Peak struct {
	BinIndex     int
	FrequencyHz  float64
	OffsetHz     float64
	PowerDbFS    float64
	SNRDb        float64
	Interpolated bool
}

func (p *PSD) PeakBin() Peak {
	if len(p.PowerDbFS) == 0 {
		return Peak{BinIndex: -1, PowerDbFS: -300}
	}
	index := 0
	for i, value := range p.PowerDbFS {
		if value > p.PowerDbFS[index] {
			index = i
		}
	}
	offsetBins := float64(index) - float64(p.FFTSize)/2
	interpolated := false
	if index > 0 && index < len(p.PowerDbFS)-1 {
		left := p.PowerDbFS[index-1]
		center := p.PowerDbFS[index]
		right := p.PowerDbFS[index+1]
		denominator := left - 2*center + right
		if denominator != 0 {
			delta := 0.5 * (left - right) / denominator
			if math.Abs(delta) <= 0.5 {
				offsetBins += delta
				interpolated = true
			}
		}
	}
	offsetHz := offsetBins * p.BinWidthHz
	return Peak{
		BinIndex:     index,
		FrequencyHz:  p.CenterFreqHz + offsetHz,
		OffsetHz:     offsetHz,
		PowerDbFS:    p.PowerDbFS[index],
		SNRDb:        p.PowerDbFS[index] - p.NoiseFloorDbFS(),
		Interpolated: interpolated,
	}
}

func (p *PSD) TotalPowerDbFS() float64 {
	total := 0.0
	for _, db := range p.PowerDbFS {
		total += math.Pow(10, db/10)
	}
	return toDb(total)
}

func (p *PSD) BandPowerDbFS(lowHz, highHz float64) float64 {
	if lowHz > highHz {
		lowHz, highHz = highHz, lowHz
	}
	total := 0.0
	counted := 0
	for i, freq := range p.FrequencyHz {
		if freq < lowHz || freq > highHz {
			continue
		}
		total += math.Pow(10, p.PowerDbFS[i]/10)
		counted++
	}
	if counted == 0 {
		return -300
	}
	return toDb(total)
}

func (p *PSD) OccupiedBandwidthHz(fraction float64) float64 {
	if fraction <= 0 || fraction >= 1 {
		fraction = 0.99
	}
	linear := make([]float64, len(p.PowerDbFS))
	total := 0.0
	for i, db := range p.PowerDbFS {
		linear[i] = math.Pow(10, db/10)
		total += linear[i]
	}
	if total <= 0 {
		return 0
	}
	target := total * (1 - fraction) / 2

	low := 0
	accumulated := 0.0
	for low < len(linear) && accumulated+linear[low] < target {
		accumulated += linear[low]
		low++
	}
	high := len(linear) - 1
	accumulated = 0
	for high >= 0 && accumulated+linear[high] < target {
		accumulated += linear[high]
		high--
	}
	if high <= low {
		return p.BinWidthHz
	}
	return float64(high-low+1) * p.BinWidthHz
}

func (p *PSD) EquivalentNoiseBandwidthHz() float64 {
	return p.windowENBW * p.BinWidthHz
}

func WaterfallRow(samples []complex128, opts PSDOptions) ([]float64, error) {
	psd, err := Welch(samples, opts)
	if err != nil {
		return nil, err
	}
	return psd.PowerDbFS, nil
}
