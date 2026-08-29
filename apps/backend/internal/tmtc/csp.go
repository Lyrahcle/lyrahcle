package tmtc

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

type CSPPacket struct {
	Priority        int    `json:"priority"`
	Source          int    `json:"source"`
	Destination     int    `json:"destination"`
	DestinationPort int    `json:"destination_port"`
	SourcePort      int    `json:"source_port"`
	Reserved        int    `json:"reserved"`
	HMAC            bool   `json:"hmac"`
	XTEA            bool   `json:"xtea"`
	RDP             bool   `json:"rdp"`
	CRC             bool   `json:"crc"`
	Payload         string `json:"payload"`
	PayloadHex      string `json:"payload_hex"`
}

const cspHeaderLen = 4

func DecodeCSPHex(packetHex string) (CSPPacket, error) {
	data, err := hex.DecodeString(strings.TrimSpace(packetHex))
	if err != nil {
		return CSPPacket{}, fmt.Errorf("tmtc: invalid hex: %w", err)
	}
	return DecodeCSP(data)
}

func DecodeCSP(data []byte) (CSPPacket, error) {
	if len(data) < cspHeaderLen {
		return CSPPacket{}, fmt.Errorf("tmtc: packet too short, need at least %d bytes for the CSP header, got %d", cspHeaderLen, len(data))
	}

	header := binary.BigEndian.Uint32(data[:cspHeaderLen])
	payload := data[cspHeaderLen:]

	return CSPPacket{
		Priority:        int(header >> 30 & 0x03),
		Destination:     int(header >> 25 & 0x1F),
		Source:          int(header >> 20 & 0x1F),
		DestinationPort: int(header >> 14 & 0x3F),
		SourcePort:      int(header >> 8 & 0x3F),
		Reserved:        int(header >> 4 & 0x0F),
		HMAC:            header>>3&0x01 == 1,
		XTEA:            header>>2&0x01 == 1,
		RDP:             header>>1&0x01 == 1,
		CRC:             header&0x01 == 1,
		Payload:         printableASCII(payload),
		PayloadHex:      hex.EncodeToString(payload),
	}, nil
}
