package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAgentHardwareCapabilitiesRoundTripTenantScoped(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	orgA := DefaultOrganizationID
	orgB := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO organizations (id, name, slug) VALUES ($1,'Other tenant',$2)`, orgB, "hardware-"+orgB.String()[:8]); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO agent_identities (agent_id, organization_id, status) VALUES ('edge-a',$1,'active'),('edge-b',$2,'active')`, orgA, orgB); err != nil {
		t.Fatal(err)
	}

	store := NewAgentIdentityStore(pool)
	report := AgentDeviceInfo{
		Driver: "rtlsdr", Hardware: "RTL-SDR Blog V4", Manufacturer: "RTL-SDR Blog", Product: "V4", Serial: "A001", RX: true,
		Capabilities: AgentDeviceCapabilities{
			Architecture: "arm64", OS: "linux", DriverVersion: "0.6.0",
			FrequencyMinHz: 24_000_000, FrequencyMaxHz: 1_766_000_000,
			SampleRatesHz: []int64{1_024_000, 2_048_000}, BandwidthsHz: []int64{1_000_000}, GainElements: []string{"TUNER"}, Channels: 1,
		},
	}
	if err := store.ReportDeviceInOrganization(ctx, orgA, "edge-a", report); err != nil {
		t.Fatalf("report device: %v", err)
	}
	if err := store.ReportDeviceInOrganization(ctx, orgB, "edge-a", report); err != ErrAgentIdentityNotFound {
		t.Fatalf("cross-tenant update err = %v, want ErrAgentIdentityNotFound", err)
	}

	itemsA, err := store.ListByOrganization(ctx, orgA)
	if err != nil {
		t.Fatal(err)
	}
	if len(itemsA) != 1 || itemsA[0].Device == nil {
		t.Fatalf("tenant A items = %+v", itemsA)
	}
	if itemsA[0].Device.Capabilities.Architecture != "arm64" || itemsA[0].Device.Compatibility != "compatible" {
		t.Fatalf("device = %+v", itemsA[0].Device)
	}
	itemsB, err := store.ListByOrganization(ctx, orgB)
	if err != nil {
		t.Fatal(err)
	}
	if len(itemsB) != 1 || itemsB[0].AgentID != "edge-b" || itemsB[0].Device != nil {
		t.Fatalf("tenant B leaked hardware: %+v", itemsB)
	}
}
