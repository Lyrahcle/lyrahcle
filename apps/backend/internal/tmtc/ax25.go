package tmtc

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type AX25Frame struct {
	Destination     string `json:"destination"`
	DestinationSSID int    `json:"destination_ssid"`
	Source          string `json:"source"`
	SourceSSID      int    `json:"source_ssid"`
	Control         byte   `json:"control"`
	PID             byte   `json:"pid"`
	Payload         string `json:"payload"`
	PayloadHex      string `json:"payload_hex"`
}

func DecodeAX25Hex(frameHex string) (AX25Frame, error) {
	data, err := hex.DecodeString(strings.TrimSpace(frameHex))
	if err != nil {
		return AX25Frame{}, fmt.Errorf("tmtc: invalid hex: %w", err)
	}
	return DecodeAX25(data)
}

func DecodeAX25(data []byte) (AX25Frame, error) {
	if len(data) < 16 {
		return AX25Frame{}, fmt.Errorf("tmtc: frame too short, need at least 16 bytes, got %d", len(data))
	}

	dest, destSSID := decodeAddress(data[0:7])
	src, srcSSID := decodeAddress(data[7:14])
	control := data[14]
	pid := data[15]
	payload := data[16:]

	return AX25Frame{
		Destination:     dest,
		DestinationSSID: destSSID,
		Source:          src,
		SourceSSID:      srcSSID,
		Control:         control,
		PID:             pid,
		Payload:         printableASCII(payload),
		PayloadHex:      hex.EncodeToString(payload),
	}, nil
}

func decodeAddress(field []byte) (callsign string, ssid int) {
	var sb strings.Builder
	for i := 0; i < 6; i++ {
		ch := field[i] >> 1
		if ch != ' ' {
			sb.WriteByte(ch)
		}
	}
	ssid = int((field[6] >> 1) & 0x0F)
	return sb.String(), ssid
}

func printableASCII(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		if b >= 32 && b < 127 {
			sb.WriteByte(b)
		} else {
			sb.WriteByte('.')
		}
	}
	return sb.String()
}
