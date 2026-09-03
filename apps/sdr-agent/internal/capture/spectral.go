package capture

import (
	"fmt"
	"sync"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/dsp"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
)

type SpectralConfig struct {
	FFTSize            int
	Window             string
	OverlapRatio       float64
	SampleCount        int
	CalibrationOffsetD float64
	ReconnectAttempts  int
}

func (c SpectralConfig) normalized() (SpectralConfig, error) {
	out := c
	if out.FFTSize == 0 {
		out.FFTSize = 4096
	}
	if !dsp.IsPowerOfTwo(out.FFTSize) {
		return out, fmt.Errorf("capture: FFT size %d is not a power of two", out.FFTSize)
	}
	if out.SampleCount == 0 {
		out.SampleCount = out.FFTSize * 8
	}
	if out.SampleCount < out.FFTSize {
		return out, fmt.Errorf("capture: sample count %d is smaller than FFT size %d", out.SampleCount, out.FFTSize)
	}
	if out.OverlapRatio < 0 || out.OverlapRatio >= 1 {
		return out, fmt.Errorf("capture: overlap ratio must be in [0,1), got %v", out.OverlapRatio)
	}
	if _, err := dsp.ParseWindowKind(out.Window); err != nil {
		return out, err
	}
	if out.ReconnectAttempts < 0 {
		out.ReconnectAttempts = 0
	}
	return out, nil
}

type reconnectable interface {
	Reconnect() error
}

type SpectralSource struct {
	mu     sync.Mutex
	device soapy.Device
	config SpectralConfig
}

func NewSpectralSource(device soapy.Device, config SpectralConfig) (*SpectralSource, error) {
	if device == nil {
		return nil, fmt.Errorf("capture: device is required")
	}
	config, err := config.normalized()
	if err != nil {
		return nil, err
	}
	if !device.IsStreaming() {
		if err := device.StartStreaming(config.SampleCount); err != nil {
			return nil, fmt.Errorf("capture: start streaming: %w", err)
		}
	}
	return &SpectralSource{device: device, config: config}, nil
}

func (s *SpectralSource) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.device.IsStreaming() {
		if err := s.device.StopStreaming(); err != nil {
			return err
		}
	}
	return s.device.Close()
}

func (s *SpectralSource) Read(frequencyHz int64) (Reading, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if frequencyHz <= 0 {
		return Reading{}, fmt.Errorf("capture: frequency must be positive, got %d", frequencyHz)
	}
	current, err := s.device.GetFrequency()
	if err != nil {
		return Reading{}, fmt.Errorf("capture: read tuned frequency: %w", err)
	}
	if current != uint64(frequencyHz) {
		if err := s.device.SetFrequency(uint64(frequencyHz)); err != nil {
			return Reading{}, fmt.Errorf("capture: retune to %d Hz: %w", frequencyHz, err)
		}
	}

	samples, err := s.readWithRetry()
	if err != nil {
		return Reading{}, err
	}

	sampleRate, err := s.device.GetSampleRate()
	if err != nil || sampleRate == 0 {
		return Reading{}, fmt.Errorf("capture: unusable sample rate: %w", err)
	}

	kind, err := dsp.ParseWindowKind(s.config.Window)
	if err != nil {
		return Reading{}, err
	}
	options := dsp.PSDOptions{
		FFTSize:      s.config.FFTSize,
		OverlapRatio: s.config.OverlapRatio,
		Window:       kind,
		SampleRateHz: float64(sampleRate),
		CenterFreqHz: float64(frequencyHz),
	}

	iq := dsp.ToComplex128(samples)
	psd, err := dsp.Welch(iq, options)
	if err != nil {
		return Reading{}, fmt.Errorf("capture: spectral estimate: %w", err)
	}
	estimate, err := dsp.EstimateCarrier(iq, options)
	if err != nil {
		return Reading{}, fmt.Errorf("capture: carrier estimate: %w", err)
	}

	capturedAt := time.Now().UTC()
	return Reading{
		FrequencyHz: frequencyHz,
		PowerDbm:    psd.TotalPowerDbFS() + s.config.CalibrationOffsetD,
		CapturedAt:  capturedAt,
		Spectral: &SpectralMetrics{
			PowerDbFS:             psd.TotalPowerDbFS(),
			NoiseFloorDbFS:        psd.NoiseFloorDbFS(),
			PeakPowerDbFS:         estimate.PowerDbFS,
			SNRDb:                 estimate.SNRDb,
			CarrierOffsetHz:       estimate.OffsetHz,
			CarrierFrequencyHz:    estimate.FrequencyHz,
			OccupiedBandwidthHz:   psd.OccupiedBandwidthHz(0.99),
			ResolutionBandwidthHz: psd.EquivalentNoiseBandwidthHz(),
			BinWidthHz:            psd.BinWidthHz,
			SampleRateHz:          float64(sampleRate),
			FFTSize:               psd.FFTSize,
			Segments:              psd.Segments,
			EstimatorMethod:       estimate.Method,
			CalibrationOffsetDb:   s.config.CalibrationOffsetD,
			Calibrated:            s.config.CalibrationOffsetD != 0,
		},
	}, nil
}

func (s *SpectralSource) readWithRetry() ([]complex64, error) {
	var lastErr error
	for attempt := 0; attempt <= s.config.ReconnectAttempts; attempt++ {
		samples, err := s.device.ReadSamples(s.config.SampleCount)
		if err == nil {
			return samples, nil
		}
		lastErr = err
		reconnector, ok := s.device.(reconnectable)
		if !ok || attempt == s.config.ReconnectAttempts {
			break
		}
		if reconnectErr := reconnector.Reconnect(); reconnectErr != nil {
			lastErr = fmt.Errorf("capture: read failed (%v) and reconnect failed: %w", err, reconnectErr)
			continue
		}
		if startErr := s.device.StartStreaming(s.config.SampleCount); startErr != nil {
			lastErr = fmt.Errorf("capture: restart streaming after reconnect: %w", startErr)
		}
	}
	return nil, fmt.Errorf("capture: read samples: %w", lastErr)
}

func (s *SpectralSource) Waterfall(rows int) ([][]float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rows <= 0 {
		return nil, fmt.Errorf("capture: rows must be positive")
	}
	sampleRate, err := s.device.GetSampleRate()
	if err != nil || sampleRate == 0 {
		return nil, fmt.Errorf("capture: unusable sample rate: %w", err)
	}
	frequency, err := s.device.GetFrequency()
	if err != nil {
		return nil, fmt.Errorf("capture: read tuned frequency: %w", err)
	}
	kind, err := dsp.ParseWindowKind(s.config.Window)
	if err != nil {
		return nil, err
	}

	out := make([][]float64, 0, rows)
	for i := 0; i < rows; i++ {
		samples, err := s.device.ReadSamples(s.config.FFTSize)
		if err != nil {
			return nil, fmt.Errorf("capture: waterfall row %d: %w", i, err)
		}
		row, err := dsp.WaterfallRow(dsp.ToComplex128(samples), dsp.PSDOptions{
			FFTSize:      s.config.FFTSize,
			OverlapRatio: s.config.OverlapRatio,
			Window:       kind,
			SampleRateHz: float64(sampleRate),
			CenterFreqHz: float64(frequency),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}
