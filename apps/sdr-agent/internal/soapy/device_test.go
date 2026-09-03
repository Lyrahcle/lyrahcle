package soapy

import (
	"testing"
)

func TestMockDeviceLifecycle(t *testing.T) {
	device := NewMockDevice()

	if device.IsStreaming() {
		t.Fatal("a fresh device must not be streaming")
	}
	if _, err := device.ReadSamples(16); err == nil {
		t.Fatal("reading before StartStreaming must fail")
	}
	if err := device.StopStreaming(); err != nil {
		t.Fatalf("StopStreaming on an idle device: %v", err)
	}

	if err := device.StartStreaming(1024); err != nil {
		t.Fatal(err)
	}
	if err := device.StartStreaming(1024); err == nil {
		t.Fatal("starting twice must fail")
	}
	samples, err := device.ReadSamples(64)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 64 {
		t.Fatalf("got %d samples, want 64", len(samples))
	}
	if err := device.Close(); err != nil {
		t.Fatal(err)
	}
	if device.IsStreaming() {
		t.Fatal("Close must stop streaming")
	}
}

func TestMockDeviceParameters(t *testing.T) {
	device := NewMockDevice()

	if err := device.SetFrequency(437_500_000); err != nil {
		t.Fatal(err)
	}
	if frequency, _ := device.GetFrequency(); frequency != 437_500_000 {
		t.Fatalf("frequency = %d", frequency)
	}
	if err := device.SetSampleRate(2_400_000); err != nil {
		t.Fatal(err)
	}
	if rate, _ := device.GetSampleRate(); rate != 2_400_000 {
		t.Fatalf("sample rate = %d", rate)
	}
	if err := device.SetGain("TUNER", 25); err != nil {
		t.Fatal(err)
	}
	if gain, _ := device.GetGain("TUNER"); gain != 25 {
		t.Fatalf("gain = %v", gain)
	}
	if err := device.SetGain("TUNER", 500); err == nil {
		t.Fatal("expected an error for an out-of-range gain")
	}
	if err := device.SetAntenna("TX/RX"); err != nil {
		t.Fatal(err)
	}
	if antenna, _ := device.GetAntenna(); antenna != "TX/RX" {
		t.Fatalf("antenna = %q", antenna)
	}
	if len(device.GetAvailableGains()) == 0 || len(device.GetAvailableAntennas()) == 0 {
		t.Fatal("the mock device must advertise gain and antenna elements")
	}

	info := device.GetDeviceInfo()
	if info == nil || info.Driver != "mock" {
		t.Fatal("expected the mock device info")
	}
	if !info.RX || info.TX {
		t.Fatal("the mock device must advertise RX only")
	}
}

func TestRTLTCPReconnectRestoresTheSession(t *testing.T) {
	server := newFakeRTLServer(t, iqPayload(4096), "RTL0")
	device, err := NewRTLTCPDevice(RTLTCPConfig{Address: server.address(), FrequencyHz: 437_500_000, SampleRateHz: 2_048_000})
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()

	reconnector, ok := device.(interface{ Reconnect() error })
	if !ok {
		t.Fatal("the rtltcp device must expose Reconnect")
	}
	if err := reconnector.Reconnect(); err == nil {
		return
	}
}

func TestRTLTCPCloseIsIdempotent(t *testing.T) {
	server := newFakeRTLServer(t, iqPayload(1024), "RTL0")
	device, err := NewRTLTCPDevice(RTLTCPConfig{Address: server.address()})
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Close(); err != nil {
		t.Fatal(err)
	}
	if err := device.Close(); err != nil {
		t.Fatalf("closing twice must be safe: %v", err)
	}
	if _, err := device.ReadSamples(16); err == nil {
		t.Fatal("reading after Close must fail")
	}
	if err := device.SetFrequency(100_000_000); err == nil {
		t.Fatal("commands after Close must fail")
	}
}

func TestRTLTCPStreamingGuards(t *testing.T) {
	server := newFakeRTLServer(t, iqPayload(2048), "RTL0")
	device, err := NewRTLTCPDevice(RTLTCPConfig{Address: server.address(), SampleRateHz: 1_024_000})
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()

	if err := device.StopStreaming(); err == nil {
		t.Fatal("stopping an idle stream must fail")
	}
	if err := device.StartStreaming(0); err != nil {
		t.Fatal(err)
	}
	if err := device.StartStreaming(1024); err == nil {
		t.Fatal("starting twice must fail")
	}
	if _, err := device.ReadSamples(0); err == nil {
		t.Fatal("a non-positive sample count must fail")
	}
	if err := device.SetSampleRate(2_400_000); err != nil {
		t.Fatal(err)
	}
	if rate, _ := device.GetSampleRate(); rate != 2_400_000 {
		t.Fatalf("sample rate = %d", rate)
	}
	if gains := device.GetAvailableGains(); len(gains) != 1 {
		t.Fatalf("gains = %v", gains)
	}
	if antennas := device.GetAvailableAntennas(); len(antennas) != 1 {
		t.Fatalf("antennas = %v", antennas)
	}
	if antenna, _ := device.GetAntenna(); antenna != "RX" {
		t.Fatalf("antenna = %q", antenna)
	}
	if gain, _ := device.GetGain("TUNER"); gain < 0 {
		t.Fatalf("gain = %v", gain)
	}
	if err := device.SetAntenna("RX"); err != nil {
		t.Fatal(err)
	}
}

func TestGainElementsReportTunerSteps(t *testing.T) {
	if elements := gainElements(0); elements[0] != "TUNER" {
		t.Fatalf("elements = %v", elements)
	}
	if elements := gainElements(29); elements[0] != "TUNER(29 steps)" {
		t.Fatalf("elements = %v", elements)
	}
}

func TestBuildRTLDeviceInfoAdaptsToTuner(t *testing.T) {
	r820 := buildRTLDeviceInfo(5, 29, RTLTCPConfig{Address: "127.0.0.1:1234"})
	if r820.Hardware != "RTL2832U/R820T" {
		t.Fatalf("hardware = %q", r820.Hardware)
	}
	if r820.Capabilities.FrequencyMinHz != 24_000_000 {
		t.Fatalf("minimum frequency = %d", r820.Capabilities.FrequencyMinHz)
	}

	e4000 := buildRTLDeviceInfo(1, 14, RTLTCPConfig{Address: "127.0.0.1:1234"})
	if e4000.Capabilities.FrequencyMaxHz != 2_200_000_000 {
		t.Fatalf("an E4000 tuner must advertise a 2.2 GHz ceiling, got %d", e4000.Capabilities.FrequencyMaxHz)
	}

	unknown := buildRTLDeviceInfo(99, 0, RTLTCPConfig{Address: "127.0.0.1:1234"})
	if unknown.Hardware != "RTL2832U/tuner-99" {
		t.Fatalf("hardware = %q", unknown.Hardware)
	}
}
