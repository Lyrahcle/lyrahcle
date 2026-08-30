package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/correlation"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

// attachCrossDomainCorrelationEvidence summarizes corroborating domains without
// overriding domain-specific severity gates. Unlinked asset alerts are context-only.
func attachCrossDomainCorrelationEvidence(
	ctx context.Context,
	incidents *store.IncidentStore,
	auditLog *store.AuditStore,
	assetAlerts *store.AssetAlertStore,
	orbitalEvents *store.OrbitalEventStore,
	incidentID uuid.UUID,
	agentID string,
	satelliteID *uuid.UUID,
	ruleCount int,
	orbital orbitalSecurityContext,
) {
	var recentAudit []store.AuditEntry
	if auditLog != nil && agentID != "" {
		var err error
		recentAudit, err = auditLog.RecentByActor(ctx, audit.ActorAgent, agentID, time.Now().UTC().Add(-24*time.Hour), 10)
		if err != nil {
			log.Printf("cross-domain: recent agent audit lookup failed for %s: %v", agentID, err)
		}
	}

	var contextualAlerts []store.AssetCVEAlert
	if assetAlerts != nil {
		var err error
		contextualAlerts, err = assetAlerts.RecentUnacknowledged(ctx, time.Now().UTC().Add(-7*24*time.Hour), 5)
		if err != nil {
			log.Printf("cross-domain: contextual asset alert lookup failed: %v", err)
		}
	}

	var operationalEvents []store.OrbitalEvent
	if orbitalEvents != nil && satelliteID != nil {
		var err error
		operationalEvents, err = orbitalEvents.RecentForSatellite(ctx, *satelliteID, time.Now().UTC().Add(-7*24*time.Hour), 20)
		if err != nil {
			log.Printf("cross-domain: orbital event lookup failed for %s: %v", satelliteID.String(), err)
		}
	}
	maneuverCount, conjunctionCount := 0, 0
	orbitalOpsConfidence := 0.0
	orbitalOpsSeverity := ""
	for _, event := range operationalEvents {
		switch event.EventType {
		case "maneuver":
			maneuverCount++
		case "conjunction":
			conjunctionCount++
		}
		if event.Confidence > orbitalOpsConfidence {
			orbitalOpsConfidence = event.Confidence
		}
		orbitalOpsSeverity = detectionHigherSeverity(orbitalOpsSeverity, event.Severity)
	}

	result := correlation.Evaluate(correlation.Input{
		DetectionRuleCount:       ruleCount,
		OrbitalCandidateSeverity: stringValue(orbital.Metadata["candidate_orbital_severity"]),
		OrbitalConfidence:        floatValue(orbital.Metadata["orbital_confidence"]),
		AgentStatus:              stringValue(orbital.Metadata["agent_status"]),
		RecentAgentAuditEvents:   len(recentAudit),
		ContextAssetAlerts:       len(contextualAlerts),
		ManeuverEvents:           maneuverCount,
		ConjunctionEvents:        conjunctionCount,
		OrbitalOpsConfidence:     orbitalOpsConfidence,
		OrbitalOpsSeverity:       orbitalOpsSeverity,
	})

	if _, err := incidents.AddEvidence(ctx, incidentID, store.CreateIncidentEvidenceInput{
		EvidenceType: "external",
		EvidenceRef:  "cross-domain-v9.4",
		Label:        "Cross-domain correlation summary",
		Metadata: map[string]interface{}{
			"engine_version":        "v9.4",
			"confidence":            result.Confidence,
			"candidate_severity":    result.CandidateSeverity,
			"corroborating_domains": result.CorroboratingDomains,
			"context_only_domains":  result.ContextOnlyDomains,
			"reasons":               result.Reasons,
			"severity_applied":      false,
			"policy":                "explainable_candidate_only",
		},
	}); err != nil {
		log.Printf("cross-domain: summary evidence write failed for incident %s: %v", incidentID, err)
	}

	if len(recentAudit) > 0 {
		items := make([]map[string]interface{}, 0, len(recentAudit))
		for _, entry := range recentAudit {
			items = append(items, map[string]interface{}{
				"id": entry.ID.String(), "action": entry.Action, "occurred_at": entry.OccurredAt,
			})
		}
		if _, err := incidents.AddEvidence(ctx, incidentID, store.CreateIncidentEvidenceInput{
			EvidenceType: "external",
			EvidenceRef:  agentID,
			Label:        "Recent agent audit activity",
			Metadata: map[string]interface{}{
				"window_hours": 24, "items": items,
			},
		}); err != nil {
			log.Printf("cross-domain: agent audit evidence write failed for incident %s: %v", incidentID, err)
		}
	}

	if len(operationalEvents) > 0 {
		items := make([]map[string]interface{}, 0, len(operationalEvents))
		for _, event := range operationalEvents {
			items = append(items, map[string]interface{}{
				"id": event.ID.String(), "event_type": event.EventType, "event_time": event.EventTime,
				"source": event.Source, "confidence": event.Confidence, "severity": event.Severity,
			})
		}
		if _, err := incidents.AddEvidence(ctx, incidentID, store.CreateIncidentEvidenceInput{
			EvidenceType: "external", EvidenceRef: "orbital-operations-v9.5", Label: "Maneuver/conjunction context",
			Metadata: map[string]interface{}{
				"candidate_only": true, "severity_applied": false, "window_days": 7, "items": items,
			},
		}); err != nil {
			log.Printf("cross-domain: orbital evidence write failed for incident %s: %v", incidentID, err)
		}
	}

	if len(contextualAlerts) > 0 {
		items := make([]map[string]interface{}, 0, len(contextualAlerts))
		for _, alert := range contextualAlerts {
			items = append(items, map[string]interface{}{
				"alert_id": alert.ID.String(), "asset_id": alert.AssetID.String(), "asset_kind": alert.AssetKind,
				"asset_value": alert.AssetValue, "cve_id": alert.CVEID, "severity": alert.Severity,
			})
		}
		if _, err := incidents.AddEvidence(ctx, incidentID, store.CreateIncidentEvidenceInput{
			EvidenceType: "external",
			EvidenceRef:  "unacknowledged-7d",
			Label:        "Recent unacknowledged asset CVE alerts",
			Metadata: map[string]interface{}{
				"context_only": true, "window_days": 7, "items": items,
			},
		}); err != nil {
			log.Printf("cross-domain: asset alert evidence write failed for incident %s: %v", incidentID, err)
		}
	}
}

func floatValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func detectionHigherSeverity(a, b string) string {
	rank := map[string]int{"": 0, "info": 1, "low": 2, "medium": 3, "high": 4, "critical": 5}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
