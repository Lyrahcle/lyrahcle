package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

const agentConfigSigningDomain = "agent_config"

func runSigningKeyRotationCheck(ctx context.Context, provider keyprovider.Provider, auditLog *store.AuditStore, now time.Time) error {
	lifecycle, ok := provider.(keyprovider.LifecycleProvider)
	if !ok || lifecycle == nil {
		return nil
	}
	policy, err := lifecycle.RotationPolicy(ctx)
	if err != nil {
		metrics.SigningKeyAutomationTotal.WithLabelValues(agentConfigSigningDomain, "policy_error").Inc()
		return err
	}
	keys, err := lifecycle.Keys(ctx)
	if err != nil {
		metrics.SigningKeyAutomationTotal.WithLabelValues(agentConfigSigningDomain, "list_error").Inc()
		return err
	}
	health := keyprovider.AssessRotation(keys, policy, now.UTC())
	metrics.RecordSigningKeyLifecycle(agentConfigSigningDomain, keys, health)
	if !policy.AutomaticPrepare || !health.RotationDue || health.StagedPending {
		metrics.SigningKeyAutomationTotal.WithLabelValues(agentConfigSigningDomain, "noop").Inc()
		return nil
	}
	metadata, _, err := lifecycle.PrepareRotation(ctx)
	if err != nil {
		if errors.Is(err, keyprovider.ErrRotationPending) {
			metrics.SigningKeyAutomationTotal.WithLabelValues(agentConfigSigningDomain, "noop").Inc()
			return nil
		}
		metrics.SigningKeyAutomationTotal.WithLabelValues(agentConfigSigningDomain, "prepare_error").Inc()
		return err
	}
	metrics.SigningKeyAutomationTotal.WithLabelValues(agentConfigSigningDomain, "prepared").Inc()
	if auditLog != nil {
		if err := auditLog.Insert(ctx, audit.ActorSystem, "signing-key-rotation", audit.ActionSigningKeyAutoPrepared, "", map[string]interface{}{
			"kid":     metadata.KID,
			"domain":  agentConfigSigningDomain,
			"trigger": "scheduled",
		}); err != nil {
			log.Printf("signing-key automation audit failed: %v", err)
		}
	}
	keys, listErr := lifecycle.Keys(ctx)
	if listErr == nil {
		metrics.RecordSigningKeyLifecycle(agentConfigSigningDomain, keys, keyprovider.AssessRotation(keys, policy, now.UTC()))
	}
	return nil
}
