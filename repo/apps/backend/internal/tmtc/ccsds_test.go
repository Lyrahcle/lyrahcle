package tmtc

import (
	"encoding/hex"
	"testing"
)

func encodeCCSDS(version int, isTC bool, secHdr bool, apid int, seqFlags int, seqCount int, payload []byte) []byte {
	var firstWord uint16
	firstWord |= uint16(version&0x07) << 13
	if isTC {
		firstWord |= 1 << 12
	}
	if secHdr {
		firstWord |= 1 << 11
	}
	firstWord |= uint16(apid & 0x07FF)

	var secondWord uint16
	secondWord |= uint16(seqFlags&0x03) << 14
	secondWord |= uint16(seqCount & 0x3FFF)

	dataLenField := uint16(len(payload) - 1)

	out := make([]byte, 6+len(payload))
	out[0] = byte(firstWord >> 8)
	out[1] = byte(firstWord)
	out[2] = byte(secondWord >> 8)
	out[3] = byte(secondWord)
	out[4] = byte(dataLenField >> 8)
	out[5] = byte(dataLenField)
	copy(out[6:], payload)
	return out
}

func TestDecodeCCSDSTelemetryRoundTrip(t *testing.T) {
	packet := encodeCCSDS(0, false, true, 100, 3, 42, []byte("TELEMETRY-OK"))
	decoded, err := DecodeCCSDSHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodeCCSDSHex returned error: %v", err)
	}

	if decoded.Version != 0 {
		t.Errorf("Version = %d, want 0", decoded.Version)
	}
	if decoded.PacketType != "TM" {
		t.Errorf("PacketType = %q, want %q", decoded.PacketType, "TM")
	}
	if !decoded.SecondaryHeaderFlag {
		t.Error("SecondaryHeaderFlag = false, want true")
	}
	if decoded.APID != 100 {
		t.Errorf("APID = %d, want 100", decoded.APID)
	}
	if decoded.SequenceFlags != "unsegmented" {
		t.Errorf("SequenceFlags = %q, want %q", decoded.SequenceFlags, "unsegmented")
	}
	if decoded.SequenceCount != 42 {
		t.Errorf("SequenceCount = %d, want 42", decoded.SequenceCount)
	}
	if decoded.DataLength != 12 {
		t.Errorf("DataLength = %d, want 12", decoded.DataLength)
	}
	if decoded.Payload != "TELEMETRY-OK" {
		t.Errorf("Payload = %q, want %q", decoded.Payload, "TELEMETRY-OK")
	}
}

func TestDecodeCCSDSTelecommand(t *testing.T) {
	packet := encodeCCSDS(0, true, false, 2047, 1, 16383, []byte("CMD"))
	decoded, err := DecodeCCSDSHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodeCCSDSHex returned error: %v", err)
	}

	if decoded.PacketType != "TC" {
		t.Errorf("PacketType = %q, want %q", decoded.PacketType, "TC")
	}
	if decoded.SecondaryHeaderFlag {
		t.Error("SecondaryHeaderFlag = true, want false")
	}
	if decoded.APID != 2047 {
		t.Errorf("APID = %d, want 2047 (max 11-bit value)", decoded.APID)
	}
	if decoded.SequenceFlags != "first" {
		t.Errorf("SequenceFlags = %q, want %q", decoded.SequenceFlags, "first")
	}
	if decoded.SequenceCount != 16383 {
		t.Errorf("SequenceCount = %d, want 16383 (max 14-bit value)", decoded.SequenceCount)
	}
}

func TestDecodeCCSDSSequenceFlagsAllValues(t *testing.T) {
	cases := []struct {
		bits int
		want string
	}{
		{0, "continuation"},
		{1, "first"},
		{2, "last"},
		{3, "unsegmented"},
	}
	for _, c := range cases {
		packet := encodeCCSDS(0, false, false, 1, c.bits, 0, []byte("x"))
		decoded, err := DecodeCCSDSHex(hex.EncodeToString(packet))
		if err != nil {
			t.Fatalf("DecodeCCSDSHex returned error: %v", err)
		}
		if decoded.SequenceFlags != c.want {
			t.Errorf("bits=%d: SequenceFlags = %q, want %q", c.bits, decoded.SequenceFlags, c.want)
		}
	}
}

func TestDecodeCCSDSTooShort(t *testing.T) {
	if _, err := DecodeCCSDSHex("0864c0"); err == nil {
		t.Error("DecodeCCSDSHex should reject a packet shorter than the 6-byte primary header")
	}
}

func TestDecodeCCSDSLengthMismatch(t *testing.T) {
	packet := encodeCCSDS(0, false, false, 1, 0, 0, []byte("hello"))
	truncated := packet[:len(packet)-2]
	if _, err := DecodeCCSDSHex(hex.EncodeToString(truncated)); err == nil {
		t.Error("DecodeCCSDSHex should reject a packet whose actual length doesn't match the declared data length")
	}
}

func TestDecodeCCSDSInvalidHex(t *testing.T) {
	if _, err := DecodeCCSDSHex("not-hex"); err == nil {
		t.Error("DecodeCCSDSHex should reject a non-hex string")
	}
}

func TestDecodeCCSDSEmptyInput(t *testing.T) {
	if _, err := DecodeCCSDSHex(""); err == nil {
		t.Error("DecodeCCSDSHex should reject an empty string")
	}
}
