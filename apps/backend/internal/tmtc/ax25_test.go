package tmtc

import (
	"encoding/hex"
	"strings"
	"testing"
)

func encodeAddress(callsign string, ssid int, last bool) []byte {
	callsign = strings.ToUpper(callsign)
	for len(callsign) < 6 {
		callsign += " "
	}
	out := make([]byte, 7)
	for i := 0; i < 6; i++ {
		out[i] = callsign[i] << 1
	}
	b := byte(ssid<<1) | 0b01100000
	if last {
		b |= 0x01
	}
	out[6] = b
	return out
}

func buildFrame(dest, src string, destSSID, srcSSID int, payload string) []byte {
	d := encodeAddress(dest, destSSID, false)
	s := encodeAddress(src, srcSSID, true)
	frame := append(append([]byte{}, d...), s...)
	frame = append(frame, 0x03, 0xF0)
	frame = append(frame, []byte(payload)...)
	return frame
}

func TestDecodeAX25RoundTrip(t *testing.T) {
	frame := buildFrame("APRS", "N0CALL", 0, 7, "=4903.50N/07201.75W-Test beacon")
	frameHex := hex.EncodeToString(frame)

	decoded, err := DecodeAX25Hex(frameHex)
	if err != nil {
		t.Fatalf("DecodeAX25Hex returned error: %v", err)
	}

	if decoded.Destination != "APRS" {
		t.Errorf("Destination = %q, want %q", decoded.Destination, "APRS")
	}
	if decoded.DestinationSSID != 0 {
		t.Errorf("DestinationSSID = %d, want 0", decoded.DestinationSSID)
	}
	if decoded.Source != "N0CALL" {
		t.Errorf("Source = %q, want %q", decoded.Source, "N0CALL")
	}
	if decoded.SourceSSID != 7 {
		t.Errorf("SourceSSID = %d, want 7", decoded.SourceSSID)
	}
	if decoded.Control != 0x03 {
		t.Errorf("Control = %#x, want 0x03", decoded.Control)
	}
	if decoded.PID != 0xF0 {
		t.Errorf("PID = %#x, want 0xF0", decoded.PID)
	}
	if decoded.Payload != "=4903.50N/07201.75W-Test beacon" {
		t.Errorf("Payload = %q, want the original beacon text", decoded.Payload)
	}
}

func TestDecodeAX25EmptyPayload(t *testing.T) {
	frame := buildFrame("CQ", "MYCALL", 0, 0, "")
	decoded, err := DecodeAX25Hex(hex.EncodeToString(frame))
	if err != nil {
		t.Fatalf("DecodeAX25Hex returned error: %v", err)
	}
	if decoded.Payload != "" {
		t.Errorf("Payload = %q, want empty", decoded.Payload)
	}
}

func TestDecodeAX25NonPrintablePayload(t *testing.T) {
	frame := buildFrame("CQ", "MYCALL", 0, 0, "")
	frame = append(frame, 0x00, 0x01, 0xFF, 0x1B)
	decoded, err := DecodeAX25Hex(hex.EncodeToString(frame))
	if err != nil {
		t.Fatalf("DecodeAX25Hex returned error: %v", err)
	}
	if decoded.Payload != "...." {
		t.Errorf("Payload = %q, want non-printable bytes replaced with dots", decoded.Payload)
	}
}

func TestDecodeAX25TooShort(t *testing.T) {
	if _, err := DecodeAX25Hex("82a0a4a6"); err == nil {
		t.Error("DecodeAX25Hex should reject a frame shorter than 16 bytes")
	}
}

func TestDecodeAX25InvalidHex(t *testing.T) {
	if _, err := DecodeAX25Hex("not-hex-at-all"); err == nil {
		t.Error("DecodeAX25Hex should reject a non-hex string")
	}
}

func TestDecodeAX25EmptyInput(t *testing.T) {
	if _, err := DecodeAX25Hex(""); err == nil {
		t.Error("DecodeAX25Hex should reject an empty string")
	}
}
