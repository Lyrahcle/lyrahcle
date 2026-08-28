package tmtc

import (
	"encoding/hex"
	"testing"
)

func encodePUSSecondaryHeader(version int, serviceType, serviceSubtype, msgCounter, destID int, payload []byte) []byte {
	out := make([]byte, 7+len(payload))
	out[0] = byte(version << 4)
	out[1] = byte(serviceType)
	out[2] = byte(serviceSubtype)
	out[3] = byte(msgCounter >> 8)
	out[4] = byte(msgCounter)
	out[5] = byte(destID >> 8)
	out[6] = byte(destID)
	copy(out[7:], payload)
	return out
}

func TestDecodePUSRoundTrip(t *testing.T) {
	secHeader := encodePUSSecondaryHeader(1, 3, 25, 4001, 7, []byte("HK-PARAM-SET-1"))
	packet := encodeCCSDS(0, false, true, 321, 3, 555, secHeader)

	decoded, err := DecodePUSHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodePUSHex returned error: %v", err)
	}

	if decoded.APID != 321 {
		t.Errorf("APID = %d, want 321", decoded.APID)
	}
	if decoded.PacketType != "TM" {
		t.Errorf("PacketType = %q, want %q", decoded.PacketType, "TM")
	}
	if decoded.SequenceCount != 555 {
		t.Errorf("SequenceCount = %d, want 555", decoded.SequenceCount)
	}
	if decoded.PUSVersion != 1 {
		t.Errorf("PUSVersion = %d, want 1", decoded.PUSVersion)
	}
	if decoded.ServiceType != 3 {
		t.Errorf("ServiceType = %d, want 3 (housekeeping)", decoded.ServiceType)
	}
	if decoded.ServiceSubtype != 25 {
		t.Errorf("ServiceSubtype = %d, want 25", decoded.ServiceSubtype)
	}
	if decoded.MessageTypeCounter != 4001 {
		t.Errorf("MessageTypeCounter = %d, want 4001", decoded.MessageTypeCounter)
	}
	if decoded.DestinationID != 7 {
		t.Errorf("DestinationID = %d, want 7", decoded.DestinationID)
	}
	if decoded.Payload != "HK-PARAM-SET-1" {
		t.Errorf("Payload = %q, want %q", decoded.Payload, "HK-PARAM-SET-1")
	}
}

func TestDecodePUSTelecommandService(t *testing.T) {
	secHeader := encodePUSSecondaryHeader(2, 8, 4, 1, 0, []byte("EXEC"))
	packet := encodeCCSDS(0, true, true, 99, 1, 1, secHeader)

	decoded, err := DecodePUSHex(hex.EncodeToString(packet))
	if err != nil {
		t.Fatalf("DecodePUSHex returned error: %v", err)
	}
	if decoded.PacketType != "TC" {
		t.Errorf("PacketType = %q, want %q", decoded.PacketType, "TC")
	}
	if decoded.ServiceType != 8 {
		t.Errorf("ServiceType = %d, want 8 (function management)", decoded.ServiceType)
	}
	if decoded.PUSVersion != 2 {
		t.Errorf("PUSVersion = %d, want 2", decoded.PUSVersion)
	}
}

func TestDecodePUSRejectsNoSecondaryHeaderFlag(t *testing.T) {
	packet := encodeCCSDS(0, false, false, 100, 0, 1, []byte("plain-ccsds-no-pus"))
	if _, err := DecodePUSHex(hex.EncodeToString(packet)); err == nil {
		t.Error("DecodePUSHex should reject a CCSDS packet with the secondary header flag unset")
	}
}

func TestDecodePUSTooShortSecondaryHeader(t *testing.T) {
	packet := encodeCCSDS(0, false, true, 100, 0, 1, []byte("ab"))
	if _, err := DecodePUSHex(hex.EncodeToString(packet)); err == nil {
		t.Error("DecodePUSHex should reject a data field too short for the PUS secondary header")
	}
}

func TestDecodePUSTooShortForCCSDS(t *testing.T) {
	if _, err := DecodePUSHex("0941"); err == nil {
		t.Error("DecodePUSHex should reject data shorter than the CCSDS primary header")
	}
}

func TestDecodePUSInvalidHex(t *testing.T) {
	if _, err := DecodePUSHex("zz"); err == nil {
		t.Error("DecodePUSHex should reject a non-hex string")
	}
}

func TestDecodePUSEmptyInput(t *testing.T) {
	if _, err := DecodePUSHex(""); err == nil {
		t.Error("DecodePUSHex should reject an empty string")
	}
}
