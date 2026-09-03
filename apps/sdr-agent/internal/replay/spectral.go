package replay

import (
	"fmt"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/dsp"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
)

type SpectralOptions struct {
	FFTSize             int
	Window              string
	OverlapRatio        float64
	WindowSamples       int
	CalibrationOffsetDb float64
}

func (o SpectralOptions) normalized(sampleCount int) (SpectralOptions, dsp.WindowKind, error) {
	out := o
	if out.FFTSize == 0 {
		out.FFTSize = 4096
	}
	if !dsp.IsPowerOfTwo(out.FFTSize) {
		return out, "", fmt.Errorf("replay: FFT size %d is not a power of two", out.FFTSize)
	}
	if out.WindowSamples <= 0 {
		out.WindowSamples = out.FFTSize * 8
	}
	if out.WindowSamples < out.FFTSize {
		return out, "", fmt.Errorf("replay: window of %d samples is smaller than the FFT size %d", out.WindowSamples, out.FFTSize)
	}
	if out.WindowSamples > sampleCount {
		out.WindowSamples = sampleCount
	}
	if out.WindowSamples < out.FFTSize {
		return out, "", fmt.Errorf("replay: capture has %d samples, fewer than the FFT size %d", sampleCount, out.FFTSize)
	}
	kind, err := dsp.ParseWindowKind(out.Window)
	if err != nil {
		return out, "", err
	}
	return out, kind, nil
}

func SpectralWindows(meta forensics.Metadata, samples []complex64, opts SpectralOptions) ([]capture.Reading, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("replay: capture has no samples")
	}
	if meta.SampleRateHz == 0 {
		return nil, fmt.Errorf("replay: capture metadata has no sample rate")
	}
	opts, kind, err := opts.normalized(len(samples))
	if err != nil {
		return nil, err
	}

	sampleRate := float64(meta.SampleRateHz)
	centerFreq := float64(meta.FrequencyHz)
	windowDuration := time.Duration(float64(opts.WindowSamples) / sampleRate * float64(time.Second))

	readings := make([]capture.Reading, 0, len(samples)/opts.WindowSamples+1)
	for start, index := 0, 0; start+opts.FFTSize <= len(samples); start, index = start+opts.WindowSamples, index+1 {
		end := start + opts.WindowSamples
		if end > len(samples) {
			end = len(samples)
		}
		iq := dsp.ToComplex128(samples[start:end])
		options := dsp.PSDOptions{
			FFTSize:      opts.FFTSize,
			OverlapRatio: opts.OverlapRatio,
			Window:       kind,
			SampleRateHz: sampleRate,
			CenterFreqHz: centerFreq,
		}
		psd, err := dsp.Welch(iq, options)
		if err != nil {
			return nil, fmt.Errorf("replay: window %d: %w", index, err)
		}
		estimate, err := dsp.EstimateCarrier(iq, options)
		if err != nil {
			return nil, fmt.Errorf("replay: window %d carrier estimate: %w", index, err)
		}
		readings = append(readings, capture.Reading{
			FrequencyHz: int64(meta.FrequencyHz),
			PowerDbm:    psd.TotalPowerDbFS() + opts.CalibrationOffsetDb,
			CapturedAt:  meta.CapturedAt.Add(time.Duration(index) * windowDuration).UTC(),
			Spectral: &capture.SpectralMetrics{
				PowerDbFS:             psd.TotalPowerDbFS(),
				NoiseFloorDbFS:        psd.NoiseFloorDbFS(),
				PeakPowerDbFS:         estimate.PowerDbFS,
				SNRDb:                 estimate.SNRDb,
				CarrierOffsetHz:       estimate.OffsetHz,
				CarrierFrequencyHz:    estimate.FrequencyHz,
				OccupiedBandwidthHz:   psd.OccupiedBandwidthHz(0.99),
				ResolutionBandwidthHz: psd.EquivalentNoiseBandwidthHz(),
				BinWidthHz:            psd.BinWidthHz,
				SampleRateHz:          sampleRate,
				FFTSize:               psd.FFTSize,
				Segments:              psd.Segments,
				EstimatorMethod:       estimate.Method,
				CalibrationOffsetDb:   opts.CalibrationOffsetDb,
				Calibrated:            opts.CalibrationOffsetDb != 0,
			},
		})
	}
	if len(readings) == 0 {
		return nil, fmt.Errorf("replay: capture of %d samples yields no complete analysis window", len(samples))
	}
	return readings, nil
}

type DopplerTrack struct {
	CapturedAt   time.Time
	OffsetHz     float64
	FrequencyHz  float64
	SNRDb        float64
	RangeRateMps float64
	Method       string
}

func DopplerTrackFromReadings(readings []capture.Reading, nominalFrequencyHz float64, minSNRDb float64) ([]DopplerTrack, error) {
	if nominalFrequencyHz <= 0 {
		return nil, fmt.Errorf("replay: nominal frequency must be positive, got %v", nominalFrequencyHz)
	}
	track := make([]DopplerTrack, 0, len(readings))
	for _, reading := range readings {
		if reading.Spectral == nil || reading.Spectral.SNRDb < minSNRDb {
			continue
		}
		rangeRate, err := dsp.RangeRateFromOffsetMps(reading.Spectral.CarrierFrequencyHz-nominalFrequencyHz, nominalFrequencyHz)
		if err != nil {
			return nil, err
		}
		track = append(track, DopplerTrack{
			CapturedAt:   reading.CapturedAt,
			OffsetHz:     reading.Spectral.CarrierFrequencyHz - nominalFrequencyHz,
			FrequencyHz:  reading.Spectral.CarrierFrequencyHz,
			SNRDb:        reading.Spectral.SNRDb,
			RangeRateMps: rangeRate,
			Method:       reading.Spectral.EstimatorMethod,
		})
	}
	if len(track) == 0 {
		return nil, fmt.Errorf("replay: no analysis window reached the %.1f dB SNR gate", minSNRDb)
	}
	return track, nil
}
