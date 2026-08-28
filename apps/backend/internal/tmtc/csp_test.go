package tmtc

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

func encodeCSP(pri, dst, src, dport, sport, rsvd int, hmac, xtea, rdp, crc bool, payload []byte) []byte {
	var header uint32
	header |= uint32(pri&0x03) << 30
	header |= uint32(dst&0x1F) << 25
	header |= uint32(src&0x1F) << 20
	header |= uint32(dport&0x3F) << 14
	header |= uint32(sport&0x3F) << 8
	header |= uint32(rsvd&0x0F) << 4
	if hmac {
		header |= 1 << 3
	}
	if xtea {
		header |= 1 << 2
	}
	if rdp {
		header |= 1 << 1
	}
	if crc {
		header |= 1
	}

	out := make([]byte, cspHeaderLen+len(payload))
	binary.BigEndian.PutUint32(out[:cspHeaderLen], header)
	copy(out[cspHeaderLen:], payload)
	return out
}

func TestDecodeCSPRoundTrip(t *testing.T) {
	packet := encodeCSP(1, 8, 1, 10, 63, 0, false, false, false, true, []byte("TEMP=21.4"))
	decoded, err := DecodeCSPHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodeCSPHex returned error: %v", err)
	}

	if decoded.Priority != 1 {
		t.Errorf("Priority = %d, want 1", decoded.Priority)
	}
	if decoded.Destination != 8 {
		t.Errorf("Destination = %d, want 8", decoded.Destination)
	}
	if decoded.Source != 1 {
		t.Errorf("Source = %d, want 1", decoded.Source)
	}
	if decoded.DestinationPort != 10 {
		t.Errorf("DestinationPort = %d, want 10", decoded.DestinationPort)
	}
	if decoded.SourcePort != 63 {
		t.Errorf("SourcePort = %d, want 63 (max 6-bit value)", decoded.SourcePort)
	}
	if decoded.HMAC {
		t.Error("HMAC = true, want false")
	}
	if !decoded.CRC {
		t.Error("CRC = false, want true")
	}
	if decoded.Payload != "TEMP=21.4" {
		t.Errorf("Payload = %q, want %q", decoded.Payload, "TEMP=21.4")
	}
}

func TestDecodeCSPAllFlags(t *testing.T) {
	packet := encodeCSP(3, 31, 31, 63, 63, 15, true, true, true, true, []byte("x"))
	decoded, err := DecodeCSPHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodeCSPHex returned error: %v", err)
	}

	if decoded.Priority != 3 {
		t.Errorf("Priority = %d, want 3 (max 2-bit value)", decoded.Priority)
	}
	if decoded.Destination != 31 {
		t.Errorf("Destination = %d, want 31 (max 5-bit value)", decoded.Destination)
	}
	if decoded.Source != 31 {
		t.Errorf("Source = %d, want 31 (max 5-bit value)", decoded.Source)
	}
	if decoded.Reserved != 15 {
		t.Errorf("Reserved = %d, want 15 (max 4-bit value)", decoded.Reserved)
	}
	if !decoded.HMAC || !decoded.XTEA || !decoded.RDP || !decoded.CRC {
		t.Error("expected all four flag bits to be set")
	}
}

func TestDecodeCSPNoFlags(t *testing.T) {
	packet := encodeCSP(0, 0, 0, 0, 0, 0, false, false, false, false, []byte("y"))
	decoded, err := DecodeCSPHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodeCSPHex returned error: %v", err)
	}
	if decoded.HMAC || decoded.XTEA || decoded.RDP || decoded.CRC {
		t.Error("expected all four flag bits to be clear")
	}
}

func TestDecodeCSPEmptyPayload(t *testing.T) {
	packet := encodeCSP(0, 1, 2, 3, 4, 0, false, false, false, false, nil)
	decoded, err := DecodeCSPHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodeCSPHex returned error: %v", err)
	}
	if decoded.Payload != "" || decoded.PayloadHex != "" {
		t.Errorf("expected empty payload, got Payload=%q PayloadHex=%q", decoded.Payload, decoded.PayloadHex)
	}
}

func TestDecodeCSPTooShort(t *testing.T) {
	if _, err := DecodeCSPHex("0102"); err == nil {
		t.Error("DecodeCSPHex should reject a packet shorter than the 4-byte header")
	}
}

func TestDecodeCSPInvalidHex(t *testing.T) {
	if _, err := DecodeCSPHex("not-hex"); err == nil {
		t.Error("DecodeCSPHex should reject a non-hex string")
	}
}

func TestDecodeCSPEmptyInput(t *testing.T) {
	if _, err := DecodeCSPHex(""); err == nil {
		t.Error("DecodeCSPHex should reject an empty string")
	}
}

func FuzzDecodeCSPHex(f *testing.F) {
	f.Add(hex.EncodeToString(encodeCSP(1, 8, 1, 10, 63, 0, false, false, false, true, []byte("seed"))))
	f.Add("")
	f.Add("00")
	f.Add("zz")

	f.Fuzz(func(t *testing.T, packetHex string) {
		_, _ = DecodeCSPHex(packetHex)
	})
}
