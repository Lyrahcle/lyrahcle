package tmtc

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type PUSPacket struct {
	APID               int    `json:"apid"`
	PacketType         string `json:"packet_type"`
	SequenceCount      int    `json:"sequence_count"`
	PUSVersion         int    `json:"pus_version"`
	ServiceType        int    `json:"service_type"`
	ServiceSubtype     int    `json:"service_subtype"`
	MessageTypeCounter int    `json:"message_type_counter"`
	DestinationID      int    `json:"destination_id"`
	Payload            string `json:"payload"`
	PayloadHex         string `json:"payload_hex"`
}

const pusSecondaryHeaderLen = 7

func DecodePUSHex(packetHex string) (PUSPacket, error) {
	data, err := hex.DecodeString(strings.TrimSpace(packetHex))
	if err != nil {
		return PUSPacket{}, fmt.Errorf("tmtc: invalid hex: %w", err)
	}
	return DecodePUS(data)
}

func DecodePUS(data []byte) (PUSPacket, error) {
	ccsds, dataField, err := decodeCCSDSHeader(data)
	if err != nil {
		return PUSPacket{}, err
	}
	if !ccsds.SecondaryHeaderFlag {
		return PUSPacket{}, fmt.Errorf("tmtc: CCSDS secondary header flag is not set; not a PUS packet")
	}
	if len(dataField) < pusSecondaryHeaderLen {
		return PUSPacket{}, fmt.Errorf("tmtc: data field too short for a PUS secondary header, need at least %d bytes, got %d", pusSecondaryHeaderLen, len(dataField))
	}

	pusVersion := int(dataField[0] >> 4)
	serviceType := int(dataField[1])
	serviceSubtype := int(dataField[2])
	messageTypeCounter := int(uint16(dataField[3])<<8 | uint16(dataField[4]))
	destinationID := int(uint16(dataField[5])<<8 | uint16(dataField[6]))

	payload := dataField[pusSecondaryHeaderLen:]

	return PUSPacket{
		APID:               ccsds.APID,
		PacketType:         ccsds.PacketType,
		SequenceCount:      ccsds.SequenceCount,
		PUSVersion:         pusVersion,
		ServiceType:        serviceType,
		ServiceSubtype:     serviceSubtype,
		MessageTypeCounter: messageTypeCounter,
		DestinationID:      destinationID,
		Payload:            printableASCII(payload),
		PayloadHex:         hex.EncodeToString(payload),
	}, nil
}
