package store

import (
	"testing"
	"time"
)

func TestFinalizeAgentDeviceInfoCompatibility(t *testing.T) {
	now := time.Date(2026, 8, 31, 6, 0, 0, 0, time.UTC)
	base := AgentDeviceInfo{
		Driver:     "rtlsdr",
		Hardware:   "RTL-SDR Blog V4",
		RX:         true,
		ReportedAt: now.Add(-time.Minute),
		Capabilities: AgentDeviceCapabilities{
			FrequencyMinHz: 24_000_000,
			FrequencyMaxHz: 1_766_000_000,
			SampleRatesHz:  []int64{1_024_000, 2_048_000},
			Channels:       1,
		},
	}

	compatible := base
	finalizeAgentDeviceInfo(&compatible, now)
	if compatible.Compatibility != "compatible" {
		t.Fatalf("compatibility = %q, want compatible (%s)", compatible.Compatibility, compatible.CompatibilityReason)
	}

	degraded := base
	degraded.Capabilities.SampleRatesHz = nil
	finalizeAgentDeviceInfo(&degraded, now)
	if degraded.Compatibility != "degraded" {
		t.Fatalf("compatibility = %q, want degraded", degraded.Compatibility)
	}

	unsupported := base
	unsupported.Driver = "mystery"
	unsupported.Hardware = "Unknown SDR"
	finalizeAgentDeviceInfo(&unsupported, now)
	if unsupported.Compatibility != "unsupported" {
		t.Fatalf("compatibility = %q, want unsupported", unsupported.Compatibility)
	}

	stale := base
	stale.ReportedAt = now.Add(-16 * time.Minute)
	finalizeAgentDeviceInfo(&stale, now)
	if stale.Compatibility != "stale" {
		t.Fatalf("compatibility = %q, want stale", stale.Compatibility)
	}
}
