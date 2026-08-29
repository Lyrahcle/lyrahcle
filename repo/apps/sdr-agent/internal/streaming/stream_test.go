package streaming

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
)

func newTestManager(t *testing.T) *StreamManager {
	t.Helper()
	device := soapy.NewMockDevice()
	if err := device.SetSampleRate(2_048_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	sm := NewStreamManager(device, StreamConfig{NumSamples: 16, BufferSize: 16, SaveCapture: true})
	if err := sm.StartCapture(); err != nil {
		t.Fatalf("StartCapture: %v", err)
	}
	t.Cleanup(func() { sm.StopCapture() })
	return sm
}

func TestReadStreamPopulatesIQData(t *testing.T) {
	sm := newTestManager(t)

	data, err := sm.ReadStream(0)
	if err != nil {
		t.Fatalf("ReadStream: %v", err)
	}
	if len(data.Samples) != 16 {
		t.Errorf("got %d samples, want 16", len(data.Samples))
	}
	if data.SampleRate != 2_048_000 {
		t.Errorf("SampleRate = %d, want 2048000", data.SampleRate)
	}
}

func TestReadStreamRequiresRunning(t *testing.T) {
	device := soapy.NewMockDevice()
	sm := NewStreamManager(device, StreamConfig{NumSamples: 16})

	if _, err := sm.ReadStream(0); err == nil {
		t.Fatal("expected an error when reading before StartCapture, got nil")
	}
}

func TestExportCaptureRequiresSamples(t *testing.T) {
	sm := newTestManager(t)
	if _, err := sm.ExportCapture("wav"); err == nil {
		t.Fatal("expected an error exporting an empty capture, got nil")
	}
}

func TestExportCaptureUnsupportedFormat(t *testing.T) {
	sm := newTestManager(t)
	if _, err := sm.ReadStream(0); err != nil {
		t.Fatalf("ReadStream: %v", err)
	}
	if _, err := sm.ExportCapture("mp3"); err == nil {
		t.Fatal("expected an error for an unsupported format, got nil")
	}
}

func TestExportWAVHeaderMatchesActualSampleRate(t *testing.T) {
	sm := newTestManager(t)
	if _, err := sm.ReadStream(0); err != nil {
		t.Fatalf("ReadStream: %v", err)
	}

	wav, err := sm.ExportCapture("wav")
	if err != nil {
		t.Fatalf("ExportCapture: %v", err)
	}

	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("missing RIFF/WAVE markers: %q / %q", wav[0:4], wav[8:12])
	}

	audioFormat := binary.LittleEndian.Uint16(wav[20:22])
	if audioFormat != 3 {
		t.Errorf("audio format = %d, want 3 (IEEE float, matches the float32 samples written)", audioFormat)
	}

	numChannels := binary.LittleEndian.Uint16(wav[22:24])
	if numChannels != 2 {
		t.Errorf("numChannels = %d, want 2", numChannels)
	}

	sampleRate := binary.LittleEndian.Uint32(wav[24:28])
	if sampleRate != 2_048_000 {
		t.Errorf("sampleRate in header = %d, want the capture's actual rate 2048000", sampleRate)
	}

	blockAlign := binary.LittleEndian.Uint16(wav[32:34])
	if blockAlign != 8 {
		t.Errorf("blockAlign = %d, want 8 (2 channels x 4 bytes)", blockAlign)
	}

	byteRate := binary.LittleEndian.Uint32(wav[28:32])
	wantByteRate := sampleRate * uint32(blockAlign)
	if byteRate != wantByteRate {
		t.Errorf("byteRate = %d, want %d (sampleRate * blockAlign)", byteRate, wantByteRate)
	}

	dataSize := binary.LittleEndian.Uint32(wav[40:44])
	wantDataSize := uint32(16 * 8) // 16 samples * 2 channels * 4 bytes
	if dataSize != wantDataSize {
		t.Errorf("dataSize = %d, want %d", dataSize, wantDataSize)
	}
	if len(wav) != 44+int(dataSize) {
		t.Errorf("total WAV length = %d, want %d (44-byte header + data)", len(wav), 44+int(dataSize))
	}
}

func TestExportIQFileRoundTripsSampleValues(t *testing.T) {
	sm := newTestManager(t)
	data, err := sm.ReadStream(0)
	if err != nil {
		t.Fatalf("ReadStream: %v", err)
	}

	raw, err := sm.ExportCapture("iqfile")
	if err != nil {
		t.Fatalf("ExportCapture: %v", err)
	}
	if len(raw) != len(data.Samples)*8 {
		t.Fatalf("got %d bytes, want %d", len(raw), len(data.Samples)*8)
	}

	for i, want := range data.Samples {
		off := i * 8
		re := math.Float32frombits(binary.LittleEndian.Uint32(raw[off:]))
		im := math.Float32frombits(binary.LittleEndian.Uint32(raw[off+4:]))
		if re != real(want) || im != imag(want) {
			t.Errorf("sample %d = (%v, %v), want (%v, %v)", i, re, im, real(want), imag(want))
		}
	}
}

func TestClearCaptureResetsState(t *testing.T) {
	sm := newTestManager(t)
	if _, err := sm.ReadStream(0); err != nil {
		t.Fatalf("ReadStream: %v", err)
	}

	sm.ClearCapture()

	if _, err := sm.ExportCapture("wav"); err == nil {
		t.Fatal("expected an error exporting after ClearCapture, got nil")
	}
	stats := sm.GetCaptureStats()
	if stats["total_samples"] != 0 {
		t.Errorf("total_samples = %v, want 0 after ClearCapture", stats["total_samples"])
	}
}
