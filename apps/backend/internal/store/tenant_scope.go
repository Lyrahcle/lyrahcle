package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantScopeStore struct {
	pool *pgxpool.Pool
}

func NewTenantScopeStore(pool *pgxpool.Pool) *TenantScopeStore {
	return &TenantScopeStore{pool: pool}
}

func (s *TenantScopeStore) AgentExists(ctx context.Context, organizationID uuid.UUID, agentID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM agent_identities WHERE agent_id=$1 AND organization_id=$2)`, agentID, organizationID).Scan(&exists)
	return exists, err
}

func (s *TenantScopeStore) AssetExists(ctx context.Context, organizationID, assetID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND organization_id=$2)`, assetID, organizationID).Scan(&exists)
	return exists, err
}

func (s *TenantScopeStore) IncidentExists(ctx context.Context, organizationID, incidentID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM incidents WHERE id=$1 AND organization_id=$2)`, incidentID, organizationID).Scan(&exists)
	return exists, err
}

func (s *TenantScopeStore) CountAgents(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM agent_identities WHERE organization_id=$1`, organizationID).Scan(&n)
	return n, err
}

func (s *TenantScopeStore) CountAssets(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM assets WHERE organization_id=$1`, organizationID).Scan(&n)
	return n, err
}

func (s *TenantScopeStore) CountIncidents(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM incidents WHERE organization_id=$1`, organizationID).Scan(&n)
	return n, err
}

func (s *TenantScopeStore) CountAuditEvents(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1`, organizationID).Scan(&n)
	return n, err
}
