package tmtc

import (
	"fmt"
	"strings"
)

const ccsdsSequenceModulo = 1 << 14

type CCSDSSequenceSample struct {
	Index         int    `json:"index"`
	APID          int    `json:"apid"`
	PacketType    string `json:"packet_type"`
	SequenceCount int    `json:"sequence_count"`
	PayloadHex    string `json:"payload_hex"`
}

type CCSDSSequenceIssue struct {
	Index           int    `json:"index"`
	APID            int    `json:"apid"`
	PacketType      string `json:"packet_type"`
	SequenceCount   int    `json:"sequence_count"`
	Kind            string `json:"kind"`
	Detail          string `json:"detail"`
	ReplayCandidate bool   `json:"replay_candidate"`
}

type CCSDSSequenceAnalysis struct {
	Samples          []CCSDSSequenceSample `json:"samples"`
	Issues           []CCSDSSequenceIssue  `json:"issues"`
	ReplayCandidates int                   `json:"replay_candidates"`
	SequenceGaps     int                   `json:"sequence_gaps"`
	OutOfOrder       int                   `json:"out_of_order"`
	Collisions       int                   `json:"collisions"`
}

type sequenceKey struct {
	APID       int
	PacketType string
}

type observedPacket struct {
	PayloadHex string
	Index      int
}

func AnalyzeCCSDSSequence(packetHexes []string) (CCSDSSequenceAnalysis, error) {
	analysis := CCSDSSequenceAnalysis{
		Samples: make([]CCSDSSequenceSample, 0, len(packetHexes)),
		Issues:  make([]CCSDSSequenceIssue, 0),
	}
	last := make(map[sequenceKey]CCSDSSequenceSample)
	seen := make(map[sequenceKey]map[int]observedPacket)

	for index, packetHex := range packetHexes {
		packet, err := DecodeCCSDSHex(strings.TrimSpace(packetHex))
		if err != nil {
			return CCSDSSequenceAnalysis{}, fmt.Errorf("tmtc: packet %d: %w", index, err)
		}
		sample := CCSDSSequenceSample{
			Index:         index,
			APID:          packet.APID,
			PacketType:    packet.PacketType,
			SequenceCount: packet.SequenceCount,
			PayloadHex:    packet.PayloadHex,
		}
		analysis.Samples = append(analysis.Samples, sample)
		key := sequenceKey{APID: packet.APID, PacketType: packet.PacketType}

		if bySequence, ok := seen[key]; ok {
			if previous, ok := bySequence[packet.SequenceCount]; ok {
				issue := CCSDSSequenceIssue{
					Index:         index,
					APID:          packet.APID,
					PacketType:    packet.PacketType,
					SequenceCount: packet.SequenceCount,
				}
				if previous.PayloadHex == packet.PayloadHex {
					issue.Kind = "replay_candidate"
					issue.ReplayCandidate = true
					issue.Detail = fmt.Sprintf("exact packet content repeats sequence count first observed at index %d", previous.Index)
					analysis.ReplayCandidates++
				} else {
					issue.Kind = "sequence_collision"
					issue.Detail = fmt.Sprintf("sequence count repeats with different payload from index %d", previous.Index)
					analysis.Collisions++
				}
				analysis.Issues = append(analysis.Issues, issue)
			}
		}

		if previous, ok := last[key]; ok && previous.SequenceCount != packet.SequenceCount {
			delta := (packet.SequenceCount - previous.SequenceCount + ccsdsSequenceModulo) % ccsdsSequenceModulo
			switch {
			case delta == 1:
			case delta > 1 && delta <= ccsdsSequenceModulo/2:
				analysis.SequenceGaps++
				analysis.Issues = append(analysis.Issues, CCSDSSequenceIssue{
					Index: index, APID: packet.APID, PacketType: packet.PacketType, SequenceCount: packet.SequenceCount,
					Kind: "sequence_gap", Detail: fmt.Sprintf("expected sequence count %d after %d", (previous.SequenceCount+1)%ccsdsSequenceModulo, previous.SequenceCount),
				})
			case delta > ccsdsSequenceModulo/2:
				analysis.OutOfOrder++
				analysis.Issues = append(analysis.Issues, CCSDSSequenceIssue{
					Index: index, APID: packet.APID, PacketType: packet.PacketType, SequenceCount: packet.SequenceCount,
					Kind: "out_of_order", Detail: fmt.Sprintf("sequence count moved backward from %d to %d", previous.SequenceCount, packet.SequenceCount),
				})
			}
		}

		if seen[key] == nil {
			seen[key] = make(map[int]observedPacket)
		}
		if _, exists := seen[key][packet.SequenceCount]; !exists {
			seen[key][packet.SequenceCount] = observedPacket{PayloadHex: packet.PayloadHex, Index: index}
		}
		last[key] = sample
	}

	return analysis, nil
}
