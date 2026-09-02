package tmtc

import (
	"encoding/hex"
	"testing"
)

func packetHex(apid, sequence int, payload string) string {
	first := uint16(apid & 0x07ff)
	second := uint16(0xc000 | (sequence & 0x3fff))
	data := []byte(payload)
	length := len(data) - 1
	packet := []byte{
		byte(first >> 8), byte(first),
		byte(second >> 8), byte(second),
		byte(uint16(length) >> 8), byte(length),
	}
	packet = append(packet, data...)
	return hex.EncodeToString(packet)
}

func TestAnalyzeCCSDSSequenceDetectsReplayCandidate(t *testing.T) {
	first := packetHex(42, 10, "CMD")
	analysis, err := AnalyzeCCSDSSequence([]string{first, packetHex(42, 11, "ACK"), first})
	if err != nil {
		t.Fatal(err)
	}
	if analysis.ReplayCandidates != 1 {
		t.Fatalf("replay candidates = %d, want 1", analysis.ReplayCandidates)
	}
	if len(analysis.Issues) == 0 || analysis.Issues[len(analysis.Issues)-1].Kind != "out_of_order" {
		t.Fatalf("issues = %#v, expected replay plus sequence ordering evidence", analysis.Issues)
	}
}

func TestAnalyzeCCSDSSequenceDistinguishesCollision(t *testing.T) {
	analysis, err := AnalyzeCCSDSSequence([]string{packetHex(7, 3, "ONE"), packetHex(7, 3, "TWO")})
	if err != nil {
		t.Fatal(err)
	}
	if analysis.ReplayCandidates != 0 || analysis.Collisions != 1 {
		t.Fatalf("analysis = %#v", analysis)
	}
}

func TestAnalyzeCCSDSSequenceDetectsGapAndWraparound(t *testing.T) {
	analysis, err := AnalyzeCCSDSSequence([]string{packetHex(2, 16383, "A"), packetHex(2, 0, "B"), packetHex(2, 2, "C")})
	if err != nil {
		t.Fatal(err)
	}
	if analysis.SequenceGaps != 1 || analysis.OutOfOrder != 0 {
		t.Fatalf("analysis = %#v", analysis)
	}
}

func TestAnalyzeCCSDSSequenceSeparatesAPIDs(t *testing.T) {
	analysis, err := AnalyzeCCSDSSequence([]string{packetHex(1, 9, "A"), packetHex(2, 20, "B"), packetHex(1, 10, "C")})
	if err != nil {
		t.Fatal(err)
	}
	if len(analysis.Issues) != 0 {
		t.Fatalf("issues = %#v, want none", analysis.Issues)
	}
}
