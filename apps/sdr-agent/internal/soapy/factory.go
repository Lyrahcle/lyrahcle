package soapy

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type DeviceSpec struct {
	URI            string
	FrequencyHz    uint64
	SampleRateHz   uint64
	GainDb         float64
	AutoGain       bool
	FreqCorrection int32
	BiasTee        bool
}

func NewDeviceFromSpec(spec DeviceSpec) (Device, error) {
	raw := strings.TrimSpace(spec.URI)
	if raw == "" || raw == "mock" || raw == "mock://" {
		return NewMockDevice(), nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("soapy: invalid device URI %q: %w", raw, err)
	}

	switch parsed.Scheme {
	case "mock":
		return NewMockDevice(), nil
	case "rtltcp":
		host := parsed.Host
		if host == "" {
			return nil, fmt.Errorf("soapy: rtltcp URI requires host:port, got %q", raw)
		}
		config := RTLTCPConfig{
			Address:        host,
			FrequencyHz:    spec.FrequencyHz,
			SampleRateHz:   spec.SampleRateHz,
			GainDb:         spec.GainDb,
			AutoGain:       spec.AutoGain,
			FreqCorrection: spec.FreqCorrection,
			BiasTee:        spec.BiasTee,
		}
		if value := parsed.Query().Get("direct_sampling"); value != "" {
			mode, err := strconv.ParseUint(value, 10, 32)
			if err != nil || mode > 2 {
				return nil, fmt.Errorf("soapy: direct_sampling must be 0, 1 or 2, got %q", value)
			}
			config.DirectSampling = uint32(mode)
		}
		if parsed.Query().Get("offset_tuning") == "1" {
			config.OffsetTuning = true
		}
		return NewRTLTCPDevice(config)
	default:
		return nil, fmt.Errorf("soapy: unsupported device scheme %q (use mock:// or rtltcp://host:port)", parsed.Scheme)
	}
}
