package soapy

import (
	"encoding/binary"
	"io"
	"math"
	"net"
	"sync"
	"testing"
	"time"
)

type fakeCommand struct {
	Code      byte
	Parameter uint32
}

type fakeRTLServer struct {
	listener net.Listener
	mu       sync.Mutex
	commands []fakeCommand
	payload  []byte
	magic    string
	done     chan struct{}
}

func newFakeRTLServer(t *testing.T, payload []byte, magic string) *fakeRTLServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &fakeRTLServer{listener: listener, payload: payload, magic: magic, done: make(chan struct{})}
	go server.serve()
	t.Cleanup(func() {
		listener.Close()
		<-server.done
	})
	return server
}

func (s *fakeRTLServer) serve() {
	defer close(s.done)
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	header := make([]byte, 12)
	copy(header, s.magic)
	binary.BigEndian.PutUint32(header[4:8], 5)
	binary.BigEndian.PutUint32(header[8:12], 29)
	if _, err := conn.Write(header); err != nil {
		return
	}
	if s.magic != "RTL0" {
		return
	}

	go func() {
		frame := make([]byte, 5)
		for {
			if _, err := io.ReadFull(conn, frame); err != nil {
				return
			}
			s.mu.Lock()
			s.commands = append(s.commands, fakeCommand{Code: frame[0], Parameter: binary.BigEndian.Uint32(frame[1:])})
			s.mu.Unlock()
		}
	}()

	for {
		if _, err := conn.Write(s.payload); err != nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *fakeRTLServer) address() string {
	return s.listener.Addr().String()
}

func (s *fakeRTLServer) recordedCommands() []fakeCommand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]fakeCommand(nil), s.commands...)
}

func iqPayload(count int) []byte {
	out := make([]byte, count*2)
	for i := 0; i < count; i++ {
		out[2*i] = byte((i * 7) % 256)
		out[2*i+1] = byte((i * 13) % 256)
	}
	return out
}

func TestRTLTCPHandshakeAndConfiguration(t *testing.T) {
	server := newFakeRTLServer(t, iqPayload(4096), "RTL0")

	device, err := NewRTLTCPDevice(RTLTCPConfig{
		Address:      server.address(),
		FrequencyHz:  437_500_000,
		SampleRateHz: 2_048_000,
		GainDb:       28.0,
	})
	if err != nil {
		t.Fatalf("NewRTLTCPDevice: %v", err)
	}
	defer device.Close()

	info := device.GetDeviceInfo()
	if info == nil {
		t.Fatal("device info is nil")
	}
	if info.Driver != "rtltcp" {
		t.Fatalf("driver = %q, want rtltcp", info.Driver)
	}
	if info.Hardware != "RTL2832U/R820T" {
		t.Fatalf("hardware = %q, want the tuner reported in the handshake", info.Hardware)
	}
	if !info.RX || info.TX {
		t.Fatal("an RTL-SDR must advertise RX only")
	}

	deadline := time.Now().Add(2 * time.Second)
	var commands []fakeCommand
	for time.Now().Before(deadline) {
		commands = server.recordedCommands()
		if len(commands) >= 4 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(commands) < 4 {
		t.Fatalf("expected at least 4 configuration commands, got %d", len(commands))
	}
	if commands[0].Code != cmdSetSampleRate || commands[0].Parameter != 2_048_000 {
		t.Fatalf("first command = %+v, want sample rate 2048000", commands[0])
	}
	if commands[1].Code != cmdSetFrequency || commands[1].Parameter != 437_500_000 {
		t.Fatalf("second command = %+v, want frequency 437500000", commands[1])
	}
	if commands[2].Code != cmdSetGainMode || commands[2].Parameter != 1 {
		t.Fatalf("third command = %+v, want manual gain mode", commands[2])
	}
	if commands[3].Code != cmdSetGain || commands[3].Parameter != 280 {
		t.Fatalf("fourth command = %+v, want a gain of 280 tenths of a dB", commands[3])
	}
}

func TestRTLTCPRejectsNonRTLServer(t *testing.T) {
	server := newFakeRTLServer(t, nil, "HTTP")
	if _, err := NewRTLTCPDevice(RTLTCPConfig{Address: server.address()}); err == nil {
		t.Fatal("expected an error when the magic bytes are not RTL0")
	}
}

func TestRTLTCPRejectsUnreachableServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()

	if _, err := NewRTLTCPDevice(RTLTCPConfig{Address: address, DialTimeout: 200 * time.Millisecond}); err == nil {
		t.Fatal("expected an error when nothing is listening")
	}
}

func TestRTLTCPReadSamplesConvertsIQ(t *testing.T) {
	server := newFakeRTLServer(t, iqPayload(8192), "RTL0")
	device, err := NewRTLTCPDevice(RTLTCPConfig{Address: server.address(), SampleRateHz: 2_048_000, FrequencyHz: 100_000_000})
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()

	if _, err := device.ReadSamples(1024); err == nil {
		t.Fatal("reading before StartStreaming must fail")
	}
	if err := device.StartStreaming(4096); err != nil {
		t.Fatal(err)
	}
	samples, err := device.ReadSamples(1024)
	if err != nil {
		t.Fatalf("ReadSamples: %v", err)
	}
	if len(samples) != 1024 {
		t.Fatalf("got %d samples, want 1024", len(samples))
	}
	for i, sample := range samples {
		if real(sample) < -1.01 || real(sample) > 1.01 || imag(sample) < -1.01 || imag(sample) > 1.01 {
			t.Fatalf("sample %d = %v is outside the normalized range", i, sample)
		}
	}
	if err := device.StopStreaming(); err != nil {
		t.Fatal(err)
	}
	if device.IsStreaming() {
		t.Fatal("device still reports streaming after StopStreaming")
	}
}

func TestConvertUint8IQNormalization(t *testing.T) {
	raw := []byte{0, 0, 255, 255, 128, 127}
	samples := ConvertUint8IQ(raw)
	if len(samples) != 3 {
		t.Fatalf("got %d samples, want 3", len(samples))
	}
	if math.Abs(float64(real(samples[0]))+1) > 0.01 {
		t.Fatalf("byte 0 should map to about -1, got %v", real(samples[0]))
	}
	if math.Abs(float64(real(samples[1]))-1) > 0.01 {
		t.Fatalf("byte 255 should map to about +1, got %v", real(samples[1]))
	}
	if math.Abs(float64(real(samples[2]))) > 0.01 || math.Abs(float64(imag(samples[2]))) > 0.01 {
		t.Fatalf("bytes near 127.5 should map to about 0, got %v", samples[2])
	}
}

func TestRTLTCPValidatesParameters(t *testing.T) {
	server := newFakeRTLServer(t, iqPayload(1024), "RTL0")
	device, err := NewRTLTCPDevice(RTLTCPConfig{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()

	if err := device.SetSampleRate(10); err == nil {
		t.Fatal("expected an error for a sample rate below the RTL2832U minimum")
	}
	if err := device.SetGain("TUNER", 120); err == nil {
		t.Fatal("expected an error for a gain above the tuner maximum")
	}
	if err := device.SetFrequency(0); err == nil {
		t.Fatal("expected an error for a zero frequency")
	}
	if err := device.SetAntenna("TX"); err == nil {
		t.Fatal("expected an error for an unsupported antenna")
	}
	if err := device.SetFrequency(145_800_000); err != nil {
		t.Fatalf("SetFrequency: %v", err)
	}
	if frequency, _ := device.GetFrequency(); frequency != 145_800_000 {
		t.Fatalf("frequency = %d, want 145800000", frequency)
	}
}

func TestNewDeviceFromSpec(t *testing.T) {
	device, err := NewDeviceFromSpec(DeviceSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if device.GetDeviceInfo().Driver != "mock" {
		t.Fatal("an empty URI must fall back to the mock device")
	}

	if _, err := NewDeviceFromSpec(DeviceSpec{URI: "hackrf://0"}); err == nil {
		t.Fatal("expected an error for an unsupported scheme")
	}
	if _, err := NewDeviceFromSpec(DeviceSpec{URI: "rtltcp://"}); err == nil {
		t.Fatal("expected an error when the rtltcp URI has no host")
	}
	if _, err := NewDeviceFromSpec(DeviceSpec{URI: "rtltcp://127.0.0.1:1?direct_sampling=9"}); err == nil {
		t.Fatal("expected an error for an invalid direct_sampling value")
	}

	server := newFakeRTLServer(t, iqPayload(1024), "RTL0")
	real, err := NewDeviceFromSpec(DeviceSpec{URI: "rtltcp://" + server.address(), FrequencyHz: 437_500_000, SampleRateHz: 2_048_000, GainDb: 30})
	if err != nil {
		t.Fatalf("NewDeviceFromSpec: %v", err)
	}
	defer real.Close()
	if real.GetDeviceInfo().Driver != "rtltcp" {
		t.Fatal("expected the rtltcp driver")
	}
}
