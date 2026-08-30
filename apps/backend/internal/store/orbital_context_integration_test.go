package store

import (
	"context"
	"testing"
	"time"
)

func TestAgentStationAndSatelliteTaggedRFReading(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	identities := NewAgentIdentityStore(pool)
	satellites := NewSatelliteStore(pool)
	rf := NewRFStore(pool)

	if _, err := pool.Exec(ctx, `INSERT INTO agent_identities (agent_id, status) VALUES ('orbital-test-agent', 'pending')`); err != nil {
		t.Fatal(err)
	}
	station := AgentStation{LatitudeDeg: 40.4168, LongitudeDeg: -3.7038, AltitudeM: 667}
	if err := identities.SetStation(ctx, "orbital-test-agent", station); err != nil {
		t.Fatal(err)
	}
	stored, err := identities.Station(ctx, "orbital-test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if stored != station {
		t.Fatalf("station = %+v, want %+v", stored, station)
	}

	line1 := "1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995"
	line2 := "2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754"
	if err := satellites.Upsert(ctx, 25544, "ISS (ZARYA)", line1, line2); err != nil {
		t.Fatal(err)
	}
	list, err := satellites.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("satellites = %v, err = %v", list, err)
	}
	reading := RFReading{AgentID: "orbital-test-agent", SatelliteID: &list[0].ID, FrequencyHz: 437500000, PowerDbm: -72, CapturedAt: time.Now().UTC()}
	if err := rf.Insert(ctx, &reading); err != nil {
		t.Fatal(err)
	}
	items, err := rf.RecentForAgent(ctx, "orbital-test-agent", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].SatelliteID == nil || *items[0].SatelliteID != list[0].ID {
		t.Fatalf("reading satellite association was not persisted: %+v", items)
	}
}
