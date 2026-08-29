package streaming

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
)

type StreamConfig struct {
	NumSamples  int
	BufferSize  int
	Timeout     time.Duration
	SaveCapture bool
	ReplayFile  string
}

type IQData struct {
	Samples    []complex64
	Timestamp  time.Time
	Frequency  uint64
	SampleRate uint64
	Metadata   map[string]string
}

type IQCapture struct {
	mu         sync.Mutex
	samples    []complex64
	sampleRate uint64
	metadata   map[string]interface{}
}

type StreamManager struct {
	device  soapy.Device
	config  StreamConfig
	mu      sync.RWMutex
	running bool
	capture *IQCapture
}

func NewStreamManager(device soapy.Device, config StreamConfig) *StreamManager {
	return &StreamManager{
		device:  device,
		config:  config,
		capture: &IQCapture{metadata: make(map[string]interface{})},
	}
}

func (sm *StreamManager) StartCapture() error {
	sm.mu.Lock()
	if sm.running {
		sm.mu.Unlock()
		return fmt.Errorf("stream: already running")
	}
	sm.running = true
	sm.mu.Unlock()

	if err := sm.device.StartStreaming(sm.config.BufferSize); err != nil {
		sm.mu.Lock()
		sm.running = false
		sm.mu.Unlock()
		return fmt.Errorf("stream: start streaming: %w", err)
	}

	return nil
}

func (sm *StreamManager) StopCapture() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return fmt.Errorf("stream: not running")
	}

	sm.running = false
	return sm.device.StopStreaming()
}

func (sm *StreamManager) ReadStream(duration time.Duration) (*IQData, error) {
	sm.mu.RLock()
	if !sm.running {
		sm.mu.RUnlock()
		return nil, fmt.Errorf("stream: not running")
	}
	sm.mu.RUnlock()

	samples, err := sm.device.ReadSamples(sm.config.NumSamples)
	if err != nil {
		return nil, fmt.Errorf("stream: read: %w", err)
	}

	freq, _ := sm.device.GetFrequency()
	rate, _ := sm.device.GetSampleRate()

	data := &IQData{
		Samples:    samples,
		Timestamp:  time.Now(),
		Frequency:  freq,
		SampleRate: rate,
		Metadata: map[string]string{
			"gain":    fmt.Sprintf("%.1f", 30.0),
			"antenna": "RX",
		},
	}

	if sm.config.SaveCapture {
		sm.captureToMemory(samples, rate)
	}

	return data, nil
}

func (sm *StreamManager) captureToMemory(samples []complex64, sampleRate uint64) {
	sm.capture.mu.Lock()
	defer sm.capture.mu.Unlock()

	sm.capture.samples = append(sm.capture.samples, samples...)
	sm.capture.sampleRate = sampleRate
	sm.capture.metadata["last_capture"] = time.Now().String()
	sm.capture.metadata["total_samples"] = len(sm.capture.samples)
}

func (sm *StreamManager) ExportCapture(format string) ([]byte, error) {
	sm.capture.mu.Lock()
	defer sm.capture.mu.Unlock()

	if len(sm.capture.samples) == 0 {
		return nil, fmt.Errorf("stream: no samples to export")
	}

	switch format {
	case "iqfile":
		return sm.exportIQFile(), nil
	case "wav":
		return sm.exportWAV(), nil
	default:
		return nil, fmt.Errorf("stream: unsupported format: %s", format)
	}
}

func (sm *StreamManager) exportIQFile() []byte {
	buf := new(bytes.Buffer)
	for _, s := range sm.capture.samples {
		binary.Write(buf, binary.LittleEndian, real(s))
		binary.Write(buf, binary.LittleEndian, imag(s))
	}
	return buf.Bytes()
}

func (sm *StreamManager) exportWAV() []byte {
	const (
		numChannels    = 2
		bitsPerSample  = 32
		bytesPerSample = bitsPerSample / 8
	)
	sampleRate := uint32(sm.capture.sampleRate)
	if sampleRate == 0 {
		sampleRate = 2_000_000
	}
	blockAlign := uint16(numChannels * bytesPerSample)
	byteRate := sampleRate * uint32(blockAlign)
	dataSize := uint32(len(sm.capture.samples) * numChannels * bytesPerSample)

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(3))
	binary.Write(buf, binary.LittleEndian, uint16(numChannels))
	binary.Write(buf, binary.LittleEndian, sampleRate)
	binary.Write(buf, binary.LittleEndian, byteRate)
	binary.Write(buf, binary.LittleEndian, blockAlign)
	binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, dataSize)

	for _, s := range sm.capture.samples {
		binary.Write(buf, binary.LittleEndian, real(s))
		binary.Write(buf, binary.LittleEndian, imag(s))
	}

	return buf.Bytes()
}

func (sm *StreamManager) GetCaptureStats() map[string]interface{} {
	sm.capture.mu.Lock()
	defer sm.capture.mu.Unlock()

	return map[string]interface{}{
		"total_samples": len(sm.capture.samples),
		"metadata":      sm.capture.metadata,
		"size_mb":       float64(len(sm.capture.samples)*8) / (1024.0 * 1024.0),
	}
}

func (sm *StreamManager) ClearCapture() {
	sm.capture.mu.Lock()
	defer sm.capture.mu.Unlock()

	sm.capture.samples = []complex64{}
	sm.capture.metadata = make(map[string]interface{})
}
