package store

import (
	"context"

	"github.com/google/uuid"
)

type TenantIsolationSnapshot struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Users          int64     `json:"users"`
	Agents         int64     `json:"agents"`
	Assets         int64     `json:"assets"`
	Incidents      int64     `json:"incidents"`
	AuditEvents    int64     `json:"audit_events"`
	Programs       int64     `json:"programs"`
	RFReadings     int64     `json:"rf_readings"`
	RFAnomalies    int64     `json:"rf_anomalies"`
	OrbitalEvents  int64     `json:"orbital_events"`
}

func (s *TenantScopeStore) Snapshot(ctx context.Context, organizationID uuid.UUID) (TenantIsolationSnapshot, error) {
	out := TenantIsolationSnapshot{OrganizationID: organizationID}
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users WHERE organization_id=$1 AND deleted_at IS NULL),
			(SELECT count(*) FROM agent_identities WHERE organization_id=$1),
			(SELECT count(*) FROM assets WHERE organization_id=$1),
			(SELECT count(*) FROM incidents WHERE organization_id=$1),
			(SELECT count(*) FROM audit_log WHERE organization_id=$1),
			(SELECT count(*) FROM programs WHERE organization_id=$1),
			(SELECT count(*) FROM rf_readings WHERE organization_id=$1),
			(SELECT count(*) FROM rf_anomalies WHERE organization_id=$1),
			(SELECT count(*) FROM orbital_events WHERE organization_id=$1)
	`, organizationID).Scan(
		&out.Users,
		&out.Agents,
		&out.Assets,
		&out.Incidents,
		&out.AuditEvents,
		&out.Programs,
		&out.RFReadings,
		&out.RFAnomalies,
		&out.OrbitalEvents,
	)
	if err != nil {
		return TenantIsolationSnapshot{}, err
	}
	return out, nil
}
