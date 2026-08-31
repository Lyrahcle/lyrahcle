package soapy

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Device interface {
	SetFrequency(freq uint64) error
	GetFrequency() (uint64, error)
	SetSampleRate(rate uint64) error
	GetSampleRate() (uint64, error)
	SetGain(gainName string, gain float64) error
	GetGain(gainName string) (float64, error)
	SetAntenna(antenna string) error
	GetAntenna() (string, error)
	GetAvailableGains() []string
	GetAvailableAntennas() []string
	StartStreaming(bufferSize int) error
	StopStreaming() error
	ReadSamples(numSamples int) ([]complex64, error)
	Close() error
	IsStreaming() bool
	GetDeviceInfo() *DeviceInfo
}

type DeviceCapabilities struct {
	Architecture    string
	OS              string
	DriverVersion   string
	FirmwareVersion string
	FrequencyMinHz  int64
	FrequencyMaxHz  int64
	SampleRatesHz   []int64
	BandwidthsHz    []int64
	GainElements    []string
	Channels        int
	GPS             bool
	PPS             bool
	TemperatureC    *float64
	SampleDrops     int64
}

type DeviceInfo struct {
	Driver       string
	Hardware     string
	Manufacturer string
	Product      string
	Serial       string
	RX           bool
	TX           bool
	Capabilities DeviceCapabilities
}

type Config struct {
	Frequency       uint64
	SampleRate      uint64
	Gain            float64
	Antenna         string
	IFGain          float64
	BBGain          float64
	LNAGain         float64
	BufferSize      int
	StreamingFormat string
}

type MockDevice struct {
	mu             sync.Mutex
	frequency      uint64
	sampleRate     uint64
	gain           float64
	antenna        string
	streaming      bool
	buffer         []complex64
	info           *DeviceInfo
	lastReadTime   time.Time
	simulatedNoise bool
}

func NewMockDevice() Device {
	return &MockDevice{
		frequency:  1e9,
		sampleRate: 2e6,
		gain:       30.0,
		antenna:    "RX",
		info: &DeviceInfo{
			Driver:       "mock",
			Hardware:     "Mock RTL-SDR",
			Manufacturer: "Lyrahcle",
			Product:      "Mock SDR Device",
			Serial:       "mock-dev-001",
			RX:           true,
			TX:           false,
			Capabilities: DeviceCapabilities{
				Architecture: runtime.GOARCH, OS: runtime.GOOS, DriverVersion: "mock", FirmwareVersion: "mock",
				FrequencyMinHz: 24_000_000, FrequencyMaxHz: 1_766_000_000,
				SampleRatesHz: []int64{1_024_000, 2_048_000, 2_400_000}, BandwidthsHz: []int64{1_000_000, 2_000_000},
				GainElements: []string{"TUNER"}, Channels: 1,
			},
		},
		buffer:         make([]complex64, 8192),
		simulatedNoise: true,
	}
}

func (d *MockDevice) SetFrequency(freq uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.frequency = freq
	return nil
}

func (d *MockDevice) GetFrequency() (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.frequency, nil
}

func (d *MockDevice) SetSampleRate(rate uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sampleRate = rate
	return nil
}

func (d *MockDevice) GetSampleRate() (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sampleRate, nil
}

func (d *MockDevice) SetGain(gainName string, gain float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if gain < 0 || gain > 50 {
		return fmt.Errorf("soapy: gain out of range: %.1f", gain)
	}
	d.gain = gain
	return nil
}

func (d *MockDevice) GetGain(gainName string) (float64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.gain, nil
}

func (d *MockDevice) SetAntenna(antenna string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.antenna = antenna
	return nil
}

func (d *MockDevice) GetAntenna() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.antenna, nil
}

func (d *MockDevice) GetAvailableGains() []string {
	return []string{"TUNER"}
}

func (d *MockDevice) GetAvailableAntennas() []string {
	return []string{"RX", "TX/RX"}
}

func (d *MockDevice) StartStreaming(bufferSize int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.streaming {
		return fmt.Errorf("soapy: already streaming")
	}
	d.streaming = true
	d.lastReadTime = time.Now()
	return nil
}

func (d *MockDevice) StopStreaming() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.streaming = false
	return nil
}

func (d *MockDevice) ReadSamples(numSamples int) ([]complex64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.streaming {
		return nil, fmt.Errorf("soapy: not streaming")
	}

	samples := make([]complex64, numSamples)
	for i := 0; i < numSamples; i++ {
		if d.simulatedNoise {
			samples[i] = complex64(complex(float32(i%10-5), float32((i*7)%10-5)))
		}
	}

	d.lastReadTime = time.Now()
	return samples, nil
}

func (d *MockDevice) IsStreaming() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.streaming
}

func (d *MockDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.streaming {
		d.streaming = false
	}
	return nil
}

func (d *MockDevice) GetDeviceInfo() *DeviceInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.info
}
