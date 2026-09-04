package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	structuredlog "github.com/mlko01/lyrahcle/apps/backend/internal/logging"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

func validateDetectionRule(name, description, metric, operator string, threshold float64, severity string, cooldownSeconds int) error {
	if err := validate.DetectionRuleName(name); err != nil {
		return err
	}
	if err := validate.DetectionRuleDescription(description); err != nil {
		return err
	}
	if err := validate.DetectionRuleMetric(metric); err != nil {
		return err
	}
	if err := validate.DetectionRuleOperator(operator); err != nil {
		return err
	}
	if err := validate.DetectionRuleThreshold(threshold); err != nil {
		return err
	}
	if err := validate.IncidentSeverity(severity); err != nil {
		return err
	}
	return validate.DetectionRuleCooldown(cooldownSeconds)
}

func validateAgentDeviceReport(info store.AgentDeviceInfo) error {
	if !info.ReportedAt.IsZero() || info.Compatibility != "" || info.CompatibilityReason != "" {
		return errors.New("reported_at and compatibility fields are read-only")
	}
	for name, value := range map[string]string{
		"driver": info.Driver, "hardware": info.Hardware, "manufacturer": info.Manufacturer,
		"product": info.Product, "serial": info.Serial,
	} {
		if len(strings.TrimSpace(value)) > 256 {
			return fmt.Errorf("%s is too long", name)
		}
	}
	c := info.Capabilities
	if c.IntegrityBaselineSHA256 != "" || c.IntegrityStatus != "" {
		return errors.New("integrity baseline and status are read-only")
	}
	for _, digest := range []string{c.IntegrityManifestSHA256, c.IntegrityMachineIDHash} {
		if digest != "" {
			if len(digest) != 64 {
				return errors.New("invalid integrity SHA-256 digest")
			}
			if _, err := hex.DecodeString(digest); err != nil {
				return errors.New("invalid integrity SHA-256 digest")
			}
		}
	}
	if len(c.IntegrityBootID) > 64 || c.IntegrityFileCount < 0 || c.IntegrityFileCount > 1024 {
		return errors.New("invalid edge integrity metadata")
	}
	if len(c.Architecture) > 64 || len(c.OS) > 64 || len(c.DriverVersion) > 128 || len(c.FirmwareVersion) > 128 {
		return errors.New("hardware capability metadata is too long")
	}
	if c.FrequencyMinHz < 0 || c.FrequencyMaxHz < 0 || (c.FrequencyMaxHz > 0 && c.FrequencyMinHz > c.FrequencyMaxHz) {
		return errors.New("invalid frequency capability range")
	}
	if c.Channels < 0 || c.Channels > 64 || c.SampleDrops < 0 || len(c.SampleRatesHz) > 64 || len(c.BandwidthsHz) > 64 || len(c.GainElements) > 64 {
		return errors.New("invalid hardware capabilities")
	}
	for _, value := range append(append([]int64{}, c.SampleRatesHz...), c.BandwidthsHz...) {
		if value <= 0 {
			return errors.New("sample rates and bandwidths must be positive")
		}
	}
	for _, name := range c.GainElements {
		if len(strings.TrimSpace(name)) > 64 {
			return errors.New("gain element name is too long")
		}
	}
	if c.TemperatureC != nil && (*c.TemperatureC < -100 || *c.TemperatureC > 250) {
		return errors.New("invalid hardware temperature")
	}
	return nil
}

func auditDetails(ctx context.Context, organizationID uuid.UUID, details map[string]interface{}) map[string]interface{} {
	enriched := make(map[string]interface{}, len(details)+3)
	for key, value := range details {
		enriched[key] = value
	}
	if organizationID != uuid.Nil {
		if _, exists := enriched["organization_id"]; !exists {
			enriched["organization_id"] = organizationID.String()
		}
	}
	if _, exists := enriched["result"]; !exists {
		enriched["result"] = "success"
	}
	if requestID := middleware.GetReqID(ctx); requestID != "" {
		if _, exists := enriched["request_id"]; !exists {
			enriched["request_id"] = requestID
		}
	}
	return enriched
}

func logAuditInOrganization(ctx context.Context, auditLog *store.AuditStore, organizationID uuid.UUID, actorType, actorID, action, ip string, details map[string]interface{}) {
	if organizationID == uuid.Nil {
		organizationID = store.DefaultOrganizationID
	}
	details = auditDetails(ctx, organizationID, details)
	if err := auditLog.InsertInOrganization(ctx, organizationID, actorType, actorID, action, ip, details); err != nil {
		structuredlog.FromContext(ctx).Error("audit_record_failed", "action", action, "actor_type", actorType, "actor_id", actorID, "tenant_id", organizationID.String(), "error", err)
	}
}

func logAudit(ctx context.Context, auditLog *store.AuditStore, actorType, actorID, action, ip string, details map[string]interface{}) {
	organizationID := store.DefaultOrganizationID
	if claims, ok := auth.FromContext(ctx); ok && claims.OrganizationID != uuid.Nil {
		organizationID = claims.OrganizationID
	} else if details != nil {
		switch value := details["organization_id"].(type) {
		case uuid.UUID:
			if value != uuid.Nil {
				organizationID = value
			}
		case string:
			if parsed, err := uuid.Parse(value); err == nil && parsed != uuid.Nil {
				organizationID = parsed
			}
		}
	}
	details = auditDetails(ctx, organizationID, details)
	if err := auditLog.InsertInOrganization(ctx, organizationID, actorType, actorID, action, ip, details); err != nil {
		structuredlog.FromContext(ctx).Error("audit_record_failed", "action", action, "actor_type", actorType, "actor_id", actorID, "error", err)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
