package tmtc

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type CCSDSPacket struct {
	Version             int    `json:"version"`
	PacketType          string `json:"packet_type"`
	SecondaryHeaderFlag bool   `json:"secondary_header_flag"`
	APID                int    `json:"apid"`
	SequenceFlags       string `json:"sequence_flags"`
	SequenceCount       int    `json:"sequence_count"`
	DataLength          int    `json:"data_length"`
	Payload             string `json:"payload"`
	PayloadHex          string `json:"payload_hex"`
}

const ccsdsPrimaryHeaderLen = 6

func DecodeCCSDSHex(packetHex string) (CCSDSPacket, error) {
	data, err := hex.DecodeString(strings.TrimSpace(packetHex))
	if err != nil {
		return CCSDSPacket{}, fmt.Errorf("tmtc: invalid hex: %w", err)
	}
	return DecodeCCSDS(data)
}

type ccsdsHeader struct {
	Version             int
	PacketType          string
	SecondaryHeaderFlag bool
	APID                int
	SequenceFlags       string
	SequenceCount       int
	DataLength          int
}

func decodeCCSDSHeader(data []byte) (ccsdsHeader, []byte, error) {
	if len(data) < ccsdsPrimaryHeaderLen {
		return ccsdsHeader{}, nil, fmt.Errorf("tmtc: packet too short, need at least %d bytes for the primary header, got %d", ccsdsPrimaryHeaderLen, len(data))
	}

	firstWord := uint16(data[0])<<8 | uint16(data[1])
	version := int(firstWord >> 13 & 0x07)
	packetType := "TM"
	if firstWord>>12&0x01 == 1 {
		packetType = "TC"
	}
	secondaryHeaderFlag := firstWord>>11&0x01 == 1
	apid := int(firstWord & 0x07FF)

	secondWord := uint16(data[2])<<8 | uint16(data[3])
	sequenceFlags := decodeSequenceFlags(secondWord >> 14 & 0x03)
	sequenceCount := int(secondWord & 0x3FFF)

	dataLength := int(uint16(data[4])<<8|uint16(data[5])) + 1

	expectedTotal := ccsdsPrimaryHeaderLen + dataLength
	if len(data) != expectedTotal {
		return ccsdsHeader{}, nil, fmt.Errorf("tmtc: declared data length implies a %d-byte packet, got %d bytes", expectedTotal, len(data))
	}

	header := ccsdsHeader{
		Version:             version,
		PacketType:          packetType,
		SecondaryHeaderFlag: secondaryHeaderFlag,
		APID:                apid,
		SequenceFlags:       sequenceFlags,
		SequenceCount:       sequenceCount,
		DataLength:          dataLength,
	}
	return header, data[ccsdsPrimaryHeaderLen:], nil
}

func DecodeCCSDS(data []byte) (CCSDSPacket, error) {
	header, payload, err := decodeCCSDSHeader(data)
	if err != nil {
		return CCSDSPacket{}, err
	}

	return CCSDSPacket{
		Version:             header.Version,
		PacketType:          header.PacketType,
		SecondaryHeaderFlag: header.SecondaryHeaderFlag,
		APID:                header.APID,
		SequenceFlags:       header.SequenceFlags,
		SequenceCount:       header.SequenceCount,
		DataLength:          header.DataLength,
		Payload:             printableASCII(payload),
		PayloadHex:          hex.EncodeToString(payload),
	}, nil
}

func decodeSequenceFlags(bits uint16) string {
	switch bits {
	case 0:
		return "continuation"
	case 1:
		return "first"
	case 2:
		return "last"
	case 3:
		return "unsegmented"
	default:
		return "unknown"
	}
}
