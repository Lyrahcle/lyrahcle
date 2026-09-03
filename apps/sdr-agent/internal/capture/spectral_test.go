package capture

import (
	"fmt"
	"math"
	"math/cmplx"
	"sync"
	"testing"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
)

type syntheticDevice struct {
	mu           sync.Mutex
	frequency    uint64
	sampleRate   uint64
	toneOffsetHz float64
	amplitude    float64
	streaming    bool
	phase        float64
	failReads    int
	reconnects   int
}

func newSyntheticDevice(sampleRate uint64, toneOffsetHz, amplitude float64) *syntheticDevice {
	return &syntheticDevice{frequency: 437_500_000, sampleRate: sampleRate, toneOffsetHz: toneOffsetHz, amplitude: amplitude}
}

func (d *syntheticDevice) SetFrequency(freq uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.frequency = freq
	return nil
}

func (d *syntheticDevice) GetFrequency() (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.frequency, nil
}

func (d *syntheticDevice) SetSampleRate(rate uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sampleRate = rate
	return nil
}

func (d *syntheticDevice) GetSampleRate() (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sampleRate, nil
}

func (d *syntheticDevice) SetGain(string, float64) error   { return nil }
func (d *syntheticDevice) GetGain(string) (float64, error) { return 30, nil }
func (d *syntheticDevice) SetAntenna(string) error         { return nil }
func (d *syntheticDevice) GetAntenna() (string, error)     { return "RX", nil }
func (d *syntheticDevice) GetAvailableGains() []string     { return []string{"TUNER"} }
func (d *syntheticDevice) GetAvailableAntennas() []string  { return []string{"RX"} }
func (d *syntheticDevice) GetDeviceInfo() *soapy.DeviceInfo {
	return &soapy.DeviceInfo{Driver: "synthetic"}
}

func (d *syntheticDevice) StartStreaming(int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.streaming = true
	return nil
}

func (d *syntheticDevice) StopStreaming() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.streaming = false
	return nil
}

func (d *syntheticDevice) IsStreaming() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.streaming
}

func (d *syntheticDevice) Close() error { return d.StopStreaming() }

func (d *syntheticDevice) Reconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.reconnects++
	d.failReads = 0
	return nil
}

func (d *syntheticDevice) ReadSamples(numSamples int) ([]complex64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.streaming {
		return nil, fmt.Errorf("synthetic: not streaming")
	}
	if d.failReads > 0 {
		d.failReads--
		return nil, fmt.Errorf("synthetic: injected read failure")
	}
	out := make([]complex64, numSamples)
	step := 2 * math.Pi * d.toneOffsetHz / float64(d.sampleRate)
	for i := range out {
		value := complex(d.amplitude, 0) * cmplx.Exp(complex(0, d.phase))
		out[i] = complex(float32(real(value)), float32(imag(value)))
		d.phase += step
	}
	return out, nil
}

func TestSpectralSourceMeasuresInjectedTone(t *testing.T) {
	device := newSyntheticDevice(2_048_000, 75_000, 0.5)
	source, err := NewSpectralSource(device, SpectralConfig{FFTSize: 4096, SampleCount: 32768, Window: "hann"})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	reading, err := source.Read(437_500_000)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if reading.Spectral == nil {
		t.Fatal("spectral metrics are missing")
	}
	if math.Abs(reading.Spectral.CarrierOffsetHz-75_000) > 100 {
		t.Fatalf("carrier offset = %.1f Hz, want 75000 Hz", reading.Spectral.CarrierOffsetHz)
	}
	if math.Abs(reading.Spectral.CarrierFrequencyHz-437_575_000) > 100 {
		t.Fatalf("carrier frequency = %.1f Hz, want 437575000 Hz", reading.Spectral.CarrierFrequencyHz)
	}
	if reading.Spectral.SNRDb < 20 {
		t.Fatalf("SNR = %.1f dB, expected a strong tone", reading.Spectral.SNRDb)
	}
	if reading.Spectral.Segments < 2 {
		t.Fatalf("segments = %d, expected Welch averaging over several segments", reading.Spectral.Segments)
	}
	if reading.Spectral.Calibrated {
		t.Fatal("readings must not claim calibration when no offset is configured")
	}
	if reading.PowerDbm != reading.Spectral.PowerDbFS {
		t.Fatal("without a calibration offset the reported power must equal the dBFS value")
	}
}

func TestSpectralSourceAppliesCalibrationOffset(t *testing.T) {
	device := newSyntheticDevice(1_024_000, 10_000, 0.5)
	source, err := NewSpectralSource(device, SpectralConfig{FFTSize: 2048, SampleCount: 16384, CalibrationOffsetD: -32.5})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	reading, err := source.Read(145_800_000)
	if err != nil {
		t.Fatal(err)
	}
	if !reading.Spectral.Calibrated {
		t.Fatal("expected the reading to be flagged as calibrated")
	}
	if math.Abs(reading.PowerDbm-(reading.Spectral.PowerDbFS-32.5)) > 1e-9 {
		t.Fatalf("power = %.3f dBm, want dBFS %.3f plus the -32.5 dB offset", reading.PowerDbm, reading.Spectral.PowerDbFS)
	}
}

func TestSpectralSourceRetunesTheDevice(t *testing.T) {
	device := newSyntheticDevice(2_048_000, 0, 0.5)
	source, err := NewSpectralSource(device, SpectralConfig{FFTSize: 2048, SampleCount: 8192})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	if _, err := source.Read(145_800_000); err != nil {
		t.Fatal(err)
	}
	frequency, _ := device.GetFrequency()
	if frequency != 145_800_000 {
		t.Fatalf("device frequency = %d, want the requested 145800000", frequency)
	}
}

func TestSpectralSourceReconnectsAfterReadFailure(t *testing.T) {
	device := newSyntheticDevice(1_024_000, 5_000, 0.5)
	device.failReads = 1
	source, err := NewSpectralSource(device, SpectralConfig{FFTSize: 2048, SampleCount: 8192, ReconnectAttempts: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	if _, err := source.Read(437_500_000); err != nil {
		t.Fatalf("Read should recover after a reconnect: %v", err)
	}
	if device.reconnects != 1 {
		t.Fatalf("reconnects = %d, want 1", device.reconnects)
	}
}

func TestSpectralSourceFailsWithoutReconnectBudget(t *testing.T) {
	device := newSyntheticDevice(1_024_000, 5_000, 0.5)
	device.failReads = 5
	source, err := NewSpectralSource(device, SpectralConfig{FFTSize: 2048, SampleCount: 8192})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	if _, err := source.Read(437_500_000); err == nil {
		t.Fatal("expected the read to fail when no reconnect attempts are configured")
	}
}

func TestSpectralConfigValidation(t *testing.T) {
	device := newSyntheticDevice(1_024_000, 0, 0.5)
	if _, err := NewSpectralSource(device, SpectralConfig{FFTSize: 3000}); err == nil {
		t.Fatal("expected an error for a non power-of-two FFT size")
	}
	if _, err := NewSpectralSource(device, SpectralConfig{FFTSize: 4096, SampleCount: 1024}); err == nil {
		t.Fatal("expected an error when the sample count is smaller than the FFT size")
	}
	if _, err := NewSpectralSource(device, SpectralConfig{Window: "kaiser"}); err == nil {
		t.Fatal("expected an error for an unsupported window")
	}
	if _, err := NewSpectralSource(nil, SpectralConfig{}); err == nil {
		t.Fatal("expected an error when no device is supplied")
	}
}

func TestSpectralSourceRejectsInvalidFrequency(t *testing.T) {
	device := newSyntheticDevice(1_024_000, 0, 0.5)
	source, err := NewSpectralSource(device, SpectralConfig{FFTSize: 2048, SampleCount: 8192})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	if _, err := source.Read(0); err == nil {
		t.Fatal("expected an error for a zero frequency")
	}
}

func TestWaterfallProducesRows(t *testing.T) {
	device := newSyntheticDevice(2_048_000, 100_000, 0.5)
	source, err := NewSpectralSource(device, SpectralConfig{FFTSize: 1024, SampleCount: 8192})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()

	rows, err := source.Waterfall(6)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("got %d rows, want 6", len(rows))
	}
	for i, row := range rows {
		if len(row) != 1024 {
			t.Fatalf("row %d has %d bins, want 1024", i, len(row))
		}
	}
	if _, err := source.Waterfall(0); err == nil {
		t.Fatal("expected an error for a non-positive row count")
	}
}
