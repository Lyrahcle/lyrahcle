package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/anomaly"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/celestrak"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/orbit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/respcache"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tlequality"
)

func syncTLEs(ctx context.Context, groups []string, satellites *store.SatelliteStore, policy tlequality.Policy) {
	defer refreshTLEMetrics(ctx, satellites, policy)
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}

		entries, err := celestrak.FetchGroup(group)
		if err != nil {
			metrics.CelestrakSyncFailuresTotal.WithLabelValues(group).Inc()
			log.Printf("tle-sync: fetch group %q: %v", group, err)
			continue
		}

		if len(entries) == 0 {
			metrics.CelestrakSyncFailuresTotal.WithLabelValues(group).Inc()
			log.Printf("tle-sync: group %q returned no entries", group)
			continue
		}

		for _, e := range entries {
			if err := satellites.Upsert(ctx, e.NoradID, e.Name, e.Line1, e.Line2); err != nil {
				metrics.CelestrakSyncFailuresTotal.WithLabelValues(group).Inc()
				log.Printf("tle-sync: upsert %d (%s): %v", e.NoradID, e.Name, err)
			}
		}

		log.Printf("tle-sync: synced %d satellites from group %q", len(entries), group)
	}
}

func refreshTLEMetrics(ctx context.Context, satellites *store.SatelliteStore, policy tlequality.Policy) {
	list, err := satellites.List(ctx)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for _, satellite := range list {
		recordTLEMetrics(satellite.NoradID, tlequality.Assess(satellite.TLEEpoch, now, policy))
	}
}

const anomalyHistoryWindow = 20

func detectRFAnomalies(ctx context.Context, anomalies *store.AnomalyStore, rules *store.DetectionRuleStore, incidents *store.IncidentStore, auditLog *store.AuditStore, assetAlerts *store.AssetAlertStore, satellites *store.SatelliteStore, identities *store.AgentIdentityStore, spaceCVEs *store.SpaceCVEStore, orbitalEvents *store.OrbitalEventStore, policy tlequality.Policy, orbitalAutoEscalation bool, reading store.RFReading, history []store.RFReading) {
	powerHistory := make([]float64, len(history))
	freqHistory := make([]float64, len(history))
	for i, h := range history {
		powerHistory[i] = h.PowerDbm
		freqHistory[i] = float64(h.FrequencyHz)
	}

	checks := []anomaly.Result{
		anomaly.Detect("power_dbm", powerHistory, reading.PowerDbm),
		anomaly.Detect("frequency_hz", freqHistory, float64(reading.FrequencyHz)),
		anomaly.DetectTrend("power_dbm", powerHistory, reading.PowerDbm),
		anomaly.DetectTrend("frequency_hz", freqHistory, float64(reading.FrequencyHz)),
	}

	for _, result := range checks {
		if !result.Anomaly {
			continue
		}
		created, err := anomalies.InsertReturning(ctx, store.RFAnomaly{
			OrganizationID: reading.OrganizationID,
			ReadingID:      reading.ID,
			AgentID:        reading.AgentID,
			SatelliteID:    reading.SatelliteID,
			Metric:         result.Metric,
			Value:          result.Value,
			BaselineMean:   result.Mean,
			BaselineStddev: result.StdDev,
			ZScore:         result.ZScore,
		})
		if err != nil {
			log.Printf("anomaly: failed to record %s anomaly for agent %s: %v", result.Metric, reading.AgentID, err)
			continue
		}
		correlateRFDetectionRules(ctx, rules, incidents, auditLog, assetAlerts, satellites, identities, spaceCVEs, orbitalEvents, policy, orbitalAutoEscalation, reading, created)
	}
}

func correlateRFDetectionRules(ctx context.Context, rules *store.DetectionRuleStore, incidents *store.IncidentStore, auditLog *store.AuditStore, assetAlerts *store.AssetAlertStore, satellites *store.SatelliteStore, identities *store.AgentIdentityStore, spaceCVEs *store.SpaceCVEStore, orbitalEvents *store.OrbitalEventStore, policy tlequality.Policy, orbitalAutoEscalation bool, reading store.RFReading, anomalyRecord store.RFAnomaly) {
	enabledRules, err := rules.ListEnabledByOrganization(ctx, anomalyRecord.OrganizationID)
	if err != nil {
		log.Printf("detection: failed to load rules: %v", err)
		return
	}

	type matchedRule struct {
		rule  detection.Rule
		match store.DetectionMatch
	}
	matched := []matchedRule{}
	autoNames := []string{}
	autoRuleCount := 0
	autoSeverity := ""
	for _, rule := range enabledRules {
		if !detection.Matches(rule, detection.Signal{Metric: anomalyRecord.Metric, ZScore: anomalyRecord.ZScore}) {
			continue
		}
		match, _, err := rules.RecordMatchIfAllowed(ctx, anomalyRecord.OrganizationID, rule, anomalyRecord.ID, anomalyRecord.AgentID, anomalyRecord.Metric, anomalyRecord.ZScore)
		if err != nil {
			log.Printf("detection: failed to record rule %s match: %v", rule.ID, err)
			continue
		}
		if match.ID == uuid.Nil {
			continue
		}
		matched = append(matched, matchedRule{rule: rule, match: match})
		if rule.AutoIncident {
			autoRuleCount++
			if len(autoNames) < 8 {
				autoNames = append(autoNames, rule.Name)
			}
			autoSeverity = detection.HigherSeverity(autoSeverity, rule.Severity)
		}
	}
	if autoRuleCount == 0 {
		return
	}

	orbitalSecurity := buildOrbitalSecurityContext(ctx, satellites, identities, spaceCVEs, policy, orbitalAutoEscalation, reading, anomalyRecord)
	if orbitalSecurity.SeverityFloor != "" {
		autoSeverity = detection.HigherSeverity(autoSeverity, orbitalSecurity.SeverityFloor)
	}

	ruleSummary := strings.Join(autoNames, ", ")
	if autoRuleCount > len(autoNames) {
		ruleSummary = fmt.Sprintf("%s (+%d more)", ruleSummary, autoRuleCount-len(autoNames))
	}
	incident, _, err := incidents.CreateWithResult(ctx, store.CreateIncidentInput{
		OrganizationID:  anomalyRecord.OrganizationID,
		Title:           fmt.Sprintf("RF detection on %s", anomalyRecord.AgentID),
		Description:     fmt.Sprintf("Matched rules: %s. Metric %s produced a z-score of %.2f.", ruleSummary, anomalyRecord.Metric, anomalyRecord.ZScore),
		Severity:        autoSeverity,
		SourceType:      "detection_rule",
		SatelliteID:     anomalyRecord.SatelliteID,
		AgentID:         anomalyRecord.AgentID,
		SourceAnomalyID: &anomalyRecord.ID,
	})
	if err != nil {
		log.Printf("detection: failed to create correlated incident for anomaly %s: %v", anomalyRecord.ID, err)
		return
	}

	matchIDs := make([]uuid.UUID, 0, len(matched))
	for _, item := range matched {
		matchIDs = append(matchIDs, item.match.ID)
		if _, err := incidents.AddEvidence(ctx, incident.ID, store.CreateIncidentEvidenceInput{
			EvidenceType: "detection_rule",
			EvidenceRef:  item.rule.ID.String(),
			Label:        item.rule.Name,
			Metadata: map[string]interface{}{
				"match_id": item.match.ID.String(),
				"metric":   anomalyRecord.Metric,
				"z_score":  anomalyRecord.ZScore,
			},
		}); err != nil {
			log.Printf("detection: failed to attach rule evidence to incident %s: %v", incident.ID, err)
		}
	}
	if _, err := incidents.AddEvidence(ctx, incident.ID, store.CreateIncidentEvidenceInput{
		EvidenceType: "rf_anomaly",
		EvidenceRef:  anomalyRecord.ID.String(),
		Label:        anomalyRecord.Metric,
	}); err != nil {
		log.Printf("detection: failed to attach anomaly evidence to incident %s: %v", incident.ID, err)
	}
	if err := rules.AttachIncidentInOrganization(ctx, anomalyRecord.OrganizationID, matchIDs, incident.ID); err != nil {
		log.Printf("detection: failed to link matches to incident %s: %v", incident.ID, err)
	}
	attachOrbitalSecurityEvidence(ctx, incidents, incident.ID.String(), orbitalSecurity)
	attachCrossDomainCorrelationEvidence(ctx, incidents, auditLog, assetAlerts, orbitalEvents, incident.ID, anomalyRecord.AgentID, anomalyRecord.SatelliteID, len(matched), orbitalSecurity)
	if auditLog != nil {
		details := map[string]interface{}{
			"organization_id": anomalyRecord.OrganizationID.String(),
			"incident_id":     incident.ID.String(),
			"reference":       incident.Reference,
			"anomaly_id":      anomalyRecord.ID.String(),
			"rule_count":      len(matched),
		}
		dedupeKey := "correlation-incident:" + incident.ID.String()
		if _, err := auditLog.InsertIdempotentInOrganization(ctx, anomalyRecord.OrganizationID, dedupeKey, audit.ActorSystem, "correlation-engine", audit.ActionIncidentCreated, "", details); err != nil {
			log.Printf("detection: failed to record idempotent incident audit for %s: %v", incident.ID, err)
		}
	}
}

type satWithPos struct {
	store.Satellite
	Position             *orbit.Position `json:"position,omitempty"`
	TLEStatus            string          `json:"tle_status"`
	TLEAgeSeconds        int64           `json:"tle_age_seconds"`
	OrbitalConfidence    string          `json:"orbital_confidence"`
	PositionEvidenceKind string          `json:"position_evidence_kind"`
	TLESource            string          `json:"tle_source"`
}

func satellitesWithPositions(ctx context.Context, satellites *store.SatelliteStore, cache *respcache.Cache, policy tlequality.Policy) ([]satWithPos, error) {
	var out []satWithPos
	if hit, _ := cache.Get(ctx, satellitesCacheKey, &out); hit {
		return out, nil
	}

	list, err := satellites.List(ctx)
	if err != nil {
		return nil, err
	}

	out = make([]satWithPos, len(list))
	workers := 8
	if len(list) < workers {
		workers = len(list)
	}
	if workers > 0 {
		jobs := make(chan int)
		var wg sync.WaitGroup
		for worker := 0; worker < workers; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for index := range jobs {
					sat := list[index]
					assessment := tlequality.Assess(sat.TLEEpoch, time.Now().UTC(), policy)
					recordTLEMetrics(sat.NoradID, assessment)
					entry := satWithPos{
						Satellite:             sat,
						TLEStatus:             string(assessment.Status),
						TLEAgeSeconds:         int64(assessment.Age.Seconds()),
						OrbitalConfidence:     assessment.Confidence,
						PositionEvidenceKind: "inferred",
						TLESource:            "celestrak_or_operator",
					}
					if assessment.Status == tlequality.StatusFresh || assessment.Status == tlequality.StatusAging {
						if pos, err := orbit.CurrentPosition(sat.TLELine1, sat.TLELine2); err == nil {
							entry.Position = &pos
						}
					}
					out[index] = entry
				}
			}()
		}
		for index := range list {
			jobs <- index
		}
		close(jobs)
		wg.Wait()
	}

	cache.Set(ctx, satellitesCacheKey, out, 5*time.Second)
	return out, nil
}

func tlePolicy(cfg config.Config) tlequality.Policy {
	return tlequality.Policy{CorrelationMaxAge: cfg.TLECorrelationMaxAge, StaleMaxAge: cfg.TLEStaleMaxAge}
}

func recordTLEMetrics(noradID int, assessment tlequality.Assessment) {
	label := strconv.Itoa(noradID)
	metrics.TLEAgeSeconds.WithLabelValues(label).Set(assessment.Age.Seconds())
	stale := 0.0
	if assessment.Status == tlequality.StatusStale || assessment.Status == tlequality.StatusUnusable {
		stale = 1
	}
	metrics.TLEStale.WithLabelValues(label).Set(stale)
	eligible := 0.0
	if assessment.EscalationAllowed {
		eligible = 1
	}
	metrics.TLECorrelationEligible.WithLabelValues(label).Set(eligible)
}
