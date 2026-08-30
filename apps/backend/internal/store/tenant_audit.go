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
	queries := []struct {
		sql  string
		dest *int64
	}{
		{`SELECT count(*) FROM users WHERE organization_id=$1 AND deleted_at IS NULL`, &out.Users},
		{`SELECT count(*) FROM agent_identities WHERE organization_id=$1`, &out.Agents},
		{`SELECT count(*) FROM assets WHERE organization_id=$1`, &out.Assets},
		{`SELECT count(*) FROM incidents WHERE organization_id=$1`, &out.Incidents},
		{`SELECT count(*) FROM audit_log WHERE organization_id=$1`, &out.AuditEvents},
		{`SELECT count(*) FROM programs WHERE organization_id=$1`, &out.Programs},
		{`SELECT count(*) FROM rf_readings WHERE organization_id=$1`, &out.RFReadings},
		{`SELECT count(*) FROM rf_anomalies WHERE organization_id=$1`, &out.RFAnomalies},
		{`SELECT count(*) FROM orbital_events WHERE organization_id=$1`, &out.OrbitalEvents},
	}
	for _, q := range queries {
		if err := s.pool.QueryRow(ctx, q.sql, organizationID).Scan(q.dest); err != nil {
			return TenantIsolationSnapshot{}, err
		}
	}
	return out, nil
}
