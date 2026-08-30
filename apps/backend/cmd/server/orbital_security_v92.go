package main

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
	"github.com/mlko01/lyrahcle/apps/backend/internal/orbitalcontext"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

type orbitalSecurityContext struct {
	Available     bool
	SeverityFloor string
	Metadata      map[string]interface{}
	ThreatCVEs    []store.SpaceCVE
}

func buildOrbitalSecurityContext(
	ctx context.Context,
	satellites *store.SatelliteStore,
	identities *store.AgentIdentityStore,
	spaceCVEs *store.SpaceCVEStore,
	reading store.RFReading,
	anomalyRecord store.RFAnomaly,
) orbitalSecurityContext {
	if reading.SatelliteID == nil {
		return orbitalSecurityContext{}
	}
	satellite, err := satellites.Get(ctx, *reading.SatelliteID)
	if err != nil {
		return orbitalSecurityContext{}
	}
	station, err := identities.Station(ctx, reading.AgentID)
	if err != nil {
		return orbitalSecurityContext{}
	}
	result, err := orbitalcontext.Evaluate(
		orbitalcontext.Station{LatitudeDeg: station.LatitudeDeg, LongitudeDeg: station.LongitudeDeg, AltitudeM: station.AltitudeM},
		satellite.TLELine1,
		satellite.TLELine2,
		reading.CapturedAt,
		float64(reading.FrequencyHz),
	)
	if err != nil {
		return orbitalSecurityContext{}
	}

	severity := ""
	if !result.Visible {
		severity = "critical"
	}
	frequencyResidual := 0.0
	frequencyTolerance := 0.0
	if anomalyRecord.Metric == "frequency_hz" || anomalyRecord.Metric == "frequency_hz_trend" {
		expected := anomalyRecord.BaselineMean + result.ExpectedDopplerHz
		frequencyResidual = math.Abs(anomalyRecord.Value - expected)
		frequencyTolerance = math.Max(2500, 3*math.Abs(anomalyRecord.BaselineStddev))
		if frequencyResidual > frequencyTolerance {
			severity = detection.HigherSeverity(severity, "high")
		}
	}

	agentStatus, err := identities.Status(ctx, reading.AgentID)
	if err == nil && agentStatus != "active" {
		severity = detection.HigherSeverity(severity, "critical")
	}

	threatCVEs, err := spaceCVEs.RecentHighSeverity(ctx, time.Now().UTC().Add(-30*24*time.Hour), 5)
	if err != nil {
		threatCVEs = nil
	}
	metadata := map[string]interface{}{
		"satellite_id":          satellite.ID.String(),
		"satellite_name":        satellite.Name,
		"norad_id":              satellite.NoradID,
		"station_latitude_deg":  station.LatitudeDeg,
		"station_longitude_deg": station.LongitudeDeg,
		"station_altitude_m":    station.AltitudeM,
		"visible":               result.Visible,
		"elevation_deg":         result.ElevationDeg,
		"range_km":              result.RangeKm,
		"range_rate_mps":        result.RangeRateMPS,
		"expected_doppler_hz":   result.ExpectedDopplerHz,
		"agent_status":          agentStatus,
	}
	if frequencyTolerance > 0 {
		metadata["frequency_residual_hz"] = frequencyResidual
		metadata["frequency_tolerance_hz"] = frequencyTolerance
	}
	return orbitalSecurityContext{Available: true, SeverityFloor: severity, Metadata: metadata, ThreatCVEs: threatCVEs}
}

func attachOrbitalSecurityEvidence(ctx context.Context, incidents *store.IncidentStore, incidentID string, security orbitalSecurityContext) {
	if !security.Available {
		return
	}
	id, err := uuid.Parse(incidentID)
	if err != nil {
		return
	}
	_, _ = incidents.AddEvidence(ctx, id, store.CreateIncidentEvidenceInput{
		EvidenceType: "satellite",
		EvidenceRef:  stringValue(security.Metadata["satellite_id"]),
		Label:        stringValue(security.Metadata["satellite_name"]),
		Metadata:     security.Metadata,
	})
	if len(security.ThreatCVEs) > 0 {
		items := make([]map[string]interface{}, 0, len(security.ThreatCVEs))
		for _, cve := range security.ThreatCVEs {
			items = append(items, map[string]interface{}{
				"cve_id":   cve.CVEID,
				"severity": cve.Severity,
				"score":    cve.Score,
				"keyword":  cve.Keyword,
			})
		}
		_, _ = incidents.AddEvidence(ctx, id, store.CreateIncidentEvidenceInput{
			EvidenceType: "external",
			EvidenceRef:  "space-threat-intel-30d",
			Label:        "Recent high-severity space CVEs",
			Metadata: map[string]interface{}{
				"context_only": true,
				"items":        items,
			},
		})
	}

}

func stringValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
