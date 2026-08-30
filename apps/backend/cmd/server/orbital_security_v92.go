package main

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/orbitalcontext"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tlequality"
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
	policy tlequality.Policy,
	autoEscalation bool,
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

	assessment := tlequality.Assess(satellite.TLEEpoch, reading.CapturedAt, policy)
	recordTLEMetrics(satellite.NoradID, tlequality.Assess(satellite.TLEEpoch, time.Now().UTC(), policy))
	metadata := map[string]interface{}{
		"satellite_id":                    satellite.ID.String(),
		"satellite_name":                  satellite.Name,
		"norad_id":                        satellite.NoradID,
		"station_latitude_deg":            station.LatitudeDeg,
		"station_longitude_deg":           station.LongitudeDeg,
		"station_altitude_m":              station.AltitudeM,
		"tle_status":                      string(assessment.Status),
		"tle_age_seconds":                 int64(assessment.Age.Seconds()),
		"orbital_confidence":              assessment.Confidence,
		"orbital_data_quality_allows":     assessment.EscalationAllowed,
		"orbital_auto_escalation_enabled": autoEscalation,
		"orbital_validation_status":       "experimental",
	}
	if satellite.TLEEpoch != nil {
		metadata["tle_epoch"] = satellite.TLEEpoch.UTC().Format(time.RFC3339Nano)
	}

	severity := ""
	agentStatus, err := identities.Status(ctx, reading.AgentID)
	if err == nil {
		metadata["agent_status"] = agentStatus
		if agentStatus != "active" {
			severity = detection.HigherSeverity(severity, "critical")
		}
	}

	candidateSeverity := ""
	if assessment.Status == tlequality.StatusFresh || assessment.Status == tlequality.StatusAging {
		result, err := orbitalcontext.Evaluate(
			orbitalcontext.Station{LatitudeDeg: station.LatitudeDeg, LongitudeDeg: station.LongitudeDeg, AltitudeM: station.AltitudeM},
			satellite.TLELine1,
			satellite.TLELine2,
			reading.CapturedAt,
			float64(reading.FrequencyHz),
		)
		if err == nil {
			metadata["visible"] = result.Visible
			metadata["elevation_deg"] = result.ElevationDeg
			metadata["range_km"] = result.RangeKm
			metadata["range_rate_mps"] = result.RangeRateMPS
			metadata["expected_doppler_hz"] = result.ExpectedDopplerHz
			if !result.Visible {
				candidateSeverity = "critical"
			}
			if anomalyRecord.Metric == "frequency_hz" || anomalyRecord.Metric == "frequency_hz_trend" {
				expected := anomalyRecord.BaselineMean + result.ExpectedDopplerHz
				frequencyResidual := math.Abs(anomalyRecord.Value - expected)
				frequencyTolerance := math.Max(2500, 3*math.Abs(anomalyRecord.BaselineStddev))
				metadata["frequency_residual_hz"] = frequencyResidual
				metadata["frequency_tolerance_hz"] = frequencyTolerance
				if frequencyResidual > frequencyTolerance {
					candidateSeverity = detection.HigherSeverity(candidateSeverity, "high")
				}
			}
		}
	}

	if candidateSeverity != "" {
		metadata["candidate_orbital_severity"] = candidateSeverity
	}
	if assessment.EscalationAllowed && autoEscalation {
		severity = detection.HigherSeverity(severity, candidateSeverity)
		metadata["orbital_escalation_applied"] = candidateSeverity != ""
	} else {
		metadata["orbital_escalation_applied"] = false
		reason := suppressionReason(assessment, autoEscalation)
		if reason != "" {
			metadata["orbital_escalation_suppressed_reason"] = reason
			metrics.OrbitalCorrelationSuppressedTotal.WithLabelValues(reason).Inc()
		}
	}

	threatCVEs, err := spaceCVEs.RecentHighSeverity(ctx, time.Now().UTC().Add(-30*24*time.Hour), 5)
	if err != nil {
		threatCVEs = nil
	}
	return orbitalSecurityContext{Available: true, SeverityFloor: severity, Metadata: metadata, ThreatCVEs: threatCVEs}
}

func suppressionReason(assessment tlequality.Assessment, autoEscalation bool) string {
	if assessment.Status == tlequality.StatusAging {
		return "tle_aging"
	}
	if assessment.Status == tlequality.StatusStale {
		return "tle_stale"
	}
	if assessment.Status == tlequality.StatusUnusable {
		return "tle_unusable"
	}
	if !autoEscalation {
		return "validation_mode"
	}
	return ""
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
