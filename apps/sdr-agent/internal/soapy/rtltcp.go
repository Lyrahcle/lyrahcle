package soapy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	cmdSetFrequency      byte = 0x01
	cmdSetSampleRate     byte = 0x02
	cmdSetGainMode       byte = 0x03
	cmdSetGain           byte = 0x04
	cmdSetFreqCorrection byte = 0x05
	cmdSetAGCMode        byte = 0x08
	cmdSetDirectSampling byte = 0x09
	cmdSetOffsetTuning   byte = 0x0a
	cmdSetBiasTee        byte = 0x0e
)

var tunerNames = map[uint32]string{
	0: "unknown",
	1: "E4000",
	2: "FC0012",
	3: "FC0013",
	4: "FC2580",
	5: "R820T",
	6: "R828D",
}

type RTLTCPConfig struct {
	Address        string
	FrequencyHz    uint64
	SampleRateHz   uint64
	GainDb         float64
	AutoGain       bool
	FreqCorrection int32
	BiasTee        bool
	DirectSampling uint32
	OffsetTuning   bool
	DialTimeout    time.Duration
	ReadTimeout    time.Duration
}

func (c RTLTCPConfig) withDefaults() RTLTCPConfig {
	out := c
	if out.Address == "" {
		out.Address = "127.0.0.1:1234"
	}
	if out.FrequencyHz == 0 {
		out.FrequencyHz = 437_500_000
	}
	if out.SampleRateHz == 0 {
		out.SampleRateHz = 2_048_000
	}
	if out.DialTimeout <= 0 {
		out.DialTimeout = 5 * time.Second
	}
	if out.ReadTimeout <= 0 {
		out.ReadTimeout = 10 * time.Second
	}
	return out
}

type RTLTCPDevice struct {
	mu          sync.Mutex
	config      RTLTCPConfig
	conn        net.Conn
	info        *DeviceInfo
	streaming   bool
	frequency   uint64
	sampleRate  uint64
	gain        float64
	antenna     string
	sampleDrops atomic.Int64
	scratch     []byte
}

func NewRTLTCPDevice(config RTLTCPConfig) (Device, error) {
	config = config.withDefaults()
	device := &RTLTCPDevice{
		config:     config,
		frequency:  config.FrequencyHz,
		sampleRate: config.SampleRateHz,
		gain:       config.GainDb,
		antenna:    "RX",
	}
	if err := device.connect(); err != nil {
		return nil, err
	}
	return device, nil
}

func (d *RTLTCPDevice) connect() error {
	conn, err := net.DialTimeout("tcp", d.config.Address, d.config.DialTimeout)
	if err != nil {
		return fmt.Errorf("rtltcp: dial %s: %w", d.config.Address, err)
	}

	header := make([]byte, 12)
	if err := conn.SetReadDeadline(time.Now().Add(d.config.DialTimeout)); err != nil {
		conn.Close()
		return fmt.Errorf("rtltcp: set handshake deadline: %w", err)
	}
	if _, err := io.ReadFull(conn, header); err != nil {
		conn.Close()
		return fmt.Errorf("rtltcp: read dongle info: %w", err)
	}
	if string(header[0:4]) != "RTL0" {
		conn.Close()
		return fmt.Errorf("rtltcp: unexpected magic %q, is %s an rtl_tcp server?", header[0:4], d.config.Address)
	}
	tunerType := binary.BigEndian.Uint32(header[4:8])
	gainCount := binary.BigEndian.Uint32(header[8:12])

	d.mu.Lock()
	d.conn = conn
	d.info = buildRTLDeviceInfo(tunerType, gainCount, d.config)
	d.mu.Unlock()

	if err := d.applyConfig(); err != nil {
		conn.Close()
		d.mu.Lock()
		d.conn = nil
		d.mu.Unlock()
		return err
	}
	return nil
}

func buildRTLDeviceInfo(tunerType, gainCount uint32, config RTLTCPConfig) *DeviceInfo {
	tuner, ok := tunerNames[tunerType]
	if !ok {
		tuner = fmt.Sprintf("tuner-%d", tunerType)
	}
	minHz := int64(24_000_000)
	maxHz := int64(1_766_000_000)
	if tuner == "E4000" {
		minHz, maxHz = 52_000_000, 2_200_000_000
	}
	return &DeviceInfo{
		Driver:       "rtltcp",
		Hardware:     "RTL2832U/" + tuner,
		Manufacturer: "Realtek",
		Product:      "RTL-SDR over rtl_tcp",
		Serial:       config.Address,
		RX:           true,
		TX:           false,
		Capabilities: DeviceCapabilities{
			Architecture:   runtime.GOARCH,
			OS:             runtime.GOOS,
			DriverVersion:  "rtl_tcp",
			FrequencyMinHz: minHz,
			FrequencyMaxHz: maxHz,
			SampleRatesHz:  []int64{240_000, 1_024_000, 1_536_000, 1_920_000, 2_048_000, 2_400_000, 2_880_000},
			BandwidthsHz:   []int64{1_000_000, 2_000_000, 2_400_000},
			GainElements:   gainElements(gainCount),
			Channels:       1,
		},
	}
}

func gainElements(gainCount uint32) []string {
	if gainCount == 0 {
		return []string{"TUNER"}
	}
	return []string{fmt.Sprintf("TUNER(%d steps)", gainCount)}
}

func (d *RTLTCPDevice) applyConfig() error {
	if err := d.sendCommand(cmdSetSampleRate, uint32(d.sampleRate)); err != nil {
		return err
	}
	if err := d.sendCommand(cmdSetFrequency, uint32(d.frequency)); err != nil {
		return err
	}
	mode := uint32(1)
	if d.config.AutoGain {
		mode = 0
	}
	if err := d.sendCommand(cmdSetGainMode, mode); err != nil {
		return err
	}
	if !d.config.AutoGain {
		if err := d.sendCommand(cmdSetGain, uint32(int32(d.gain*10))); err != nil {
			return err
		}
	}
	if err := d.sendCommand(cmdSetAGCMode, 0); err != nil {
		return err
	}
	if d.config.FreqCorrection != 0 {
		if err := d.sendCommand(cmdSetFreqCorrection, uint32(d.config.FreqCorrection)); err != nil {
			return err
		}
	}
	if d.config.DirectSampling != 0 {
		if err := d.sendCommand(cmdSetDirectSampling, d.config.DirectSampling); err != nil {
			return err
		}
	}
	if d.config.OffsetTuning {
		if err := d.sendCommand(cmdSetOffsetTuning, 1); err != nil {
			return err
		}
	}
	if d.config.BiasTee {
		if err := d.sendCommand(cmdSetBiasTee, 1); err != nil {
			return err
		}
	}
	return nil
}

func (d *RTLTCPDevice) sendCommand(command byte, parameter uint32) error {
	d.mu.Lock()
	conn := d.conn
	d.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("rtltcp: not connected")
	}
	frame := make([]byte, 5)
	frame[0] = command
	binary.BigEndian.PutUint32(frame[1:], parameter)
	if err := conn.SetWriteDeadline(time.Now().Add(d.config.DialTimeout)); err != nil {
		return fmt.Errorf("rtltcp: set write deadline: %w", err)
	}
	if _, err := conn.Write(frame); err != nil {
		return fmt.Errorf("rtltcp: send command 0x%02x: %w", command, err)
	}
	return nil
}

func (d *RTLTCPDevice) SetFrequency(freq uint64) error {
	if freq == 0 {
		return fmt.Errorf("rtltcp: frequency must be positive")
	}
	if err := d.sendCommand(cmdSetFrequency, uint32(freq)); err != nil {
		return err
	}
	d.mu.Lock()
	d.frequency = freq
	d.mu.Unlock()
	return nil
}

func (d *RTLTCPDevice) GetFrequency() (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.frequency, nil
}

func (d *RTLTCPDevice) SetSampleRate(rate uint64) error {
	if rate < 225_001 || rate > 3_200_000 {
		return fmt.Errorf("rtltcp: sample rate %d out of supported range", rate)
	}
	if err := d.sendCommand(cmdSetSampleRate, uint32(rate)); err != nil {
		return err
	}
	d.mu.Lock()
	d.sampleRate = rate
	d.mu.Unlock()
	return nil
}

func (d *RTLTCPDevice) GetSampleRate() (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sampleRate, nil
}

func (d *RTLTCPDevice) SetGain(gainName string, gain float64) error {
	if gain < 0 || gain > 49.6 {
		return fmt.Errorf("rtltcp: gain %.1f dB out of range for an R820T tuner", gain)
	}
	if err := d.sendCommand(cmdSetGainMode, 1); err != nil {
		return err
	}
	if err := d.sendCommand(cmdSetGain, uint32(int32(gain*10))); err != nil {
		return err
	}
	d.mu.Lock()
	d.gain = gain
	d.mu.Unlock()
	return nil
}

func (d *RTLTCPDevice) GetGain(gainName string) (float64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.gain, nil
}

func (d *RTLTCPDevice) SetAntenna(antenna string) error {
	if antenna != "RX" {
		return fmt.Errorf("rtltcp: RTL-SDR exposes a single RX antenna port")
	}
	return nil
}

func (d *RTLTCPDevice) GetAntenna() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.antenna, nil
}

func (d *RTLTCPDevice) GetAvailableGains() []string {
	return []string{"TUNER"}
}

func (d *RTLTCPDevice) GetAvailableAntennas() []string {
	return []string{"RX"}
}

func (d *RTLTCPDevice) StartStreaming(bufferSize int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.streaming {
		return fmt.Errorf("rtltcp: already streaming")
	}
	if d.conn == nil {
		return fmt.Errorf("rtltcp: not connected")
	}
	if bufferSize <= 0 {
		bufferSize = 16384
	}
	d.scratch = make([]byte, bufferSize*2)
	d.streaming = true
	return nil
}

func (d *RTLTCPDevice) StopStreaming() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.streaming {
		return fmt.Errorf("rtltcp: not streaming")
	}
	d.streaming = false
	return nil
}

func (d *RTLTCPDevice) ReadSamples(numSamples int) ([]complex64, error) {
	if numSamples <= 0 {
		return nil, fmt.Errorf("rtltcp: numSamples must be positive")
	}

	d.mu.Lock()
	if !d.streaming {
		d.mu.Unlock()
		return nil, fmt.Errorf("rtltcp: not streaming")
	}
	conn := d.conn
	readTimeout := d.config.ReadTimeout
	d.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("rtltcp: not connected")
	}

	raw := make([]byte, numSamples*2)
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return nil, fmt.Errorf("rtltcp: set read deadline: %w", err)
	}
	read, err := io.ReadFull(conn, raw)
	if err != nil {
		d.sampleDrops.Add(int64(numSamples - read/2))
		return nil, fmt.Errorf("rtltcp: read %d samples: %w", numSamples, err)
	}

	return ConvertUint8IQ(raw), nil
}

func ConvertUint8IQ(raw []byte) []complex64 {
	count := len(raw) / 2
	out := make([]complex64, count)
	for i := 0; i < count; i++ {
		inPhase := (float32(raw[2*i]) - 127.5) / 127.5
		quadrature := (float32(raw[2*i+1]) - 127.5) / 127.5
		out[i] = complex(inPhase, quadrature)
	}
	return out
}

func (d *RTLTCPDevice) IsStreaming() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.streaming
}

func (d *RTLTCPDevice) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.streaming = false
	if d.conn == nil {
		return nil
	}
	err := d.conn.Close()
	d.conn = nil
	if err != nil {
		return fmt.Errorf("rtltcp: close: %w", err)
	}
	return nil
}

func (d *RTLTCPDevice) GetDeviceInfo() *DeviceInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.info == nil {
		return nil
	}
	snapshot := *d.info
	snapshot.Capabilities.SampleDrops = d.sampleDrops.Load()
	return &snapshot
}

func (d *RTLTCPDevice) Reconnect() error {
	if err := d.Close(); err != nil {
		return err
	}
	return d.connect()
}
