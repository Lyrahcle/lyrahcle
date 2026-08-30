package main

import (
	"context"
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
	incidentID uuid.UUID,
	agentID string,
	ruleCount int,
	orbital orbitalSecurityContext,
) {
	var recentAudit []store.AuditEntry
	if auditLog != nil && agentID != "" {
		recentAudit, _ = auditLog.RecentByActor(ctx, audit.ActorAgent, agentID, time.Now().UTC().Add(-24*time.Hour), 10)
	}

	var contextualAlerts []store.AssetCVEAlert
	if assetAlerts != nil {
		contextualAlerts, _ = assetAlerts.RecentUnacknowledged(ctx, time.Now().UTC().Add(-7*24*time.Hour), 5)
	}

	result := correlation.Evaluate(correlation.Input{
		DetectionRuleCount:       ruleCount,
		OrbitalCandidateSeverity: stringValue(orbital.Metadata["candidate_orbital_severity"]),
		OrbitalConfidence:        floatValue(orbital.Metadata["orbital_confidence"]),
		AgentStatus:              stringValue(orbital.Metadata["agent_status"]),
		RecentAgentAuditEvents:   len(recentAudit),
		ContextAssetAlerts:       len(contextualAlerts),
	})

	_, _ = incidents.AddEvidence(ctx, incidentID, store.CreateIncidentEvidenceInput{
		EvidenceType: "correlation",
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
	})

	if len(recentAudit) > 0 {
		items := make([]map[string]interface{}, 0, len(recentAudit))
		for _, entry := range recentAudit {
			items = append(items, map[string]interface{}{
				"id": entry.ID.String(), "action": entry.Action, "occurred_at": entry.OccurredAt,
			})
		}
		_, _ = incidents.AddEvidence(ctx, incidentID, store.CreateIncidentEvidenceInput{
			EvidenceType: "audit",
			EvidenceRef:  agentID,
			Label:        "Recent agent audit activity",
			Metadata: map[string]interface{}{
				"window_hours": 24, "items": items,
			},
		})
	}

	if len(contextualAlerts) > 0 {
		items := make([]map[string]interface{}, 0, len(contextualAlerts))
		for _, alert := range contextualAlerts {
			items = append(items, map[string]interface{}{
				"alert_id": alert.ID.String(), "asset_id": alert.AssetID.String(), "asset_kind": alert.AssetKind,
				"asset_value": alert.AssetValue, "cve_id": alert.CVEID, "severity": alert.Severity,
			})
		}
		_, _ = incidents.AddEvidence(ctx, incidentID, store.CreateIncidentEvidenceInput{
			EvidenceType: "asset_alert_context",
			EvidenceRef:  "unacknowledged-7d",
			Label:        "Recent unacknowledged asset CVE alerts",
			Metadata: map[string]interface{}{
				"context_only": true, "window_days": 7, "items": items,
			},
		})
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
