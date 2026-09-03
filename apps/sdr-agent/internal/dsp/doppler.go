package dsp

import (
	"fmt"
	"math"
	"math/cmplx"
)

const SpeedOfLightMps = 299792458.0

type CarrierEstimate struct {
	CoarseOffsetHz float64
	FineOffsetHz   float64
	OffsetHz       float64
	FrequencyHz    float64
	PowerDbFS      float64
	NoiseFloorDbFS float64
	SNRDb          float64
	BinWidthHz     float64
	Method         string
	Converged      bool
}

func EstimateCarrier(samples []complex128, opts PSDOptions) (CarrierEstimate, error) {
	psd, err := Welch(samples, opts)
	if err != nil {
		return CarrierEstimate{}, err
	}
	peak := psd.PeakBin()
	estimate := CarrierEstimate{
		CoarseOffsetHz: peak.OffsetHz,
		OffsetHz:       peak.OffsetHz,
		FrequencyHz:    psd.CenterFreqHz + peak.OffsetHz,
		PowerDbFS:      peak.PowerDbFS,
		NoiseFloorDbFS: psd.NoiseFloorDbFS(),
		SNRDb:          peak.SNRDb,
		BinWidthHz:     psd.BinWidthHz,
		Method:         "psd-peak",
	}
	if peak.Interpolated {
		estimate.Method = "psd-peak-interpolated"
	}

	if estimate.SNRDb < 6 {
		return estimate, nil
	}

	baseband := MixDown(samples, peak.OffsetHz, psd.SampleRateHz)
	residual, ok := KayFrequencyOffset(baseband, psd.SampleRateHz)
	if !ok {
		return estimate, nil
	}
	if math.Abs(residual) > psd.BinWidthHz {
		return estimate, nil
	}
	estimate.FineOffsetHz = residual
	estimate.OffsetHz = peak.OffsetHz + residual
	estimate.FrequencyHz = psd.CenterFreqHz + estimate.OffsetHz
	estimate.Method = "psd-peak+kay"
	estimate.Converged = true
	return estimate, nil
}

func MixDown(samples []complex128, offsetHz, sampleRateHz float64) []complex128 {
	if sampleRateHz <= 0 {
		return samples
	}
	out := make([]complex128, len(samples))
	step := -2 * math.Pi * offsetHz / sampleRateHz
	for i, s := range samples {
		phase := step * float64(i)
		out[i] = s * cmplx.Exp(complex(0, phase))
	}
	return out
}

func KayFrequencyOffset(samples []complex128, sampleRateHz float64) (float64, bool) {
	n := len(samples)
	if n < 3 || sampleRateHz <= 0 {
		return 0, false
	}
	count := float64(n)
	denominator := count * (count*count - 1)
	if denominator == 0 {
		return 0, false
	}
	var accumulated float64
	for i := 0; i < n-1; i++ {
		product := samples[i+1] * cmplx.Conj(samples[i])
		if product == 0 {
			return 0, false
		}
		weight := 6 * float64(i+1) * (count - float64(i+1)) / denominator
		accumulated += weight * cmplx.Phase(product)
	}
	if math.IsNaN(accumulated) || math.IsInf(accumulated, 0) {
		return 0, false
	}
	return accumulated * sampleRateHz / (2 * math.Pi), true
}

func RangeRateFromOffsetMps(offsetHz, nominalFrequencyHz float64) (float64, error) {
	if nominalFrequencyHz <= 0 {
		return 0, fmt.Errorf("dsp: nominal frequency must be positive, got %v", nominalFrequencyHz)
	}
	return -SpeedOfLightMps * offsetHz / nominalFrequencyHz, nil
}

func OffsetFromRangeRateHz(rangeRateMps, nominalFrequencyHz float64) (float64, error) {
	if nominalFrequencyHz <= 0 {
		return 0, fmt.Errorf("dsp: nominal frequency must be positive, got %v", nominalFrequencyHz)
	}
	return -nominalFrequencyHz * rangeRateMps / SpeedOfLightMps, nil
}

type DopplerResidual struct {
	ObservedOffsetHz  float64
	PredictedOffsetHz float64
	ResidualHz        float64
	ResidualBins      float64
	ToleranceHz       float64
	WithinTolerance   bool
	SNRDb             float64
}

func CompareWithPrediction(estimate CarrierEstimate, predictedOffsetHz, toleranceHz float64) DopplerResidual {
	if toleranceHz <= 0 {
		toleranceHz = estimate.BinWidthHz
	}
	residual := estimate.OffsetHz - predictedOffsetHz
	bins := 0.0
	if estimate.BinWidthHz > 0 {
		bins = residual / estimate.BinWidthHz
	}
	return DopplerResidual{
		ObservedOffsetHz:  estimate.OffsetHz,
		PredictedOffsetHz: predictedOffsetHz,
		ResidualHz:        residual,
		ResidualBins:      bins,
		ToleranceHz:       toleranceHz,
		WithinTolerance:   math.Abs(residual) <= toleranceHz,
		SNRDb:             estimate.SNRDb,
	}
}
