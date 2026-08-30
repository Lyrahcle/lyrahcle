package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Asset struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ProgramID      uuid.UUID `json:"program_id"`
	Kind           string    `json:"kind"`
	Value          string    `json:"value"`
}

type AssetStore struct {
	pool *pgxpool.Pool
}

func NewAssetStore(pool *pgxpool.Pool) *AssetStore {
	return &AssetStore{pool: pool}
}

func (s *AssetStore) List(ctx context.Context, limit, offset int) (Page[Asset], error) {
	limit, offset = ClampPageParams(limit, offset)

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM assets`).Scan(&total); err != nil {
		return Page[Asset]{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, program_id, kind, value FROM assets
		ORDER BY last_seen DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return Page[Asset]{}, err
	}
	defer rows.Close()

	items := []Asset{}
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.OrganizationID, &a.ProgramID, &a.Kind, &a.Value); err != nil {
			return Page[Asset]{}, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return Page[Asset]{}, err
	}

	return Page[Asset]{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *AssetStore) Upsert(ctx context.Context, programID uuid.UUID, kind, value string) error {
	return s.UpsertInOrganization(ctx, DefaultOrganizationID, programID, kind, value)
}

func (s *AssetStore) ListByKinds(ctx context.Context, kinds []string) ([]Asset, error) {
	if len(kinds) == 0 {
		return nil, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, program_id, kind, value FROM assets
		WHERE kind = ANY($1)
		ORDER BY last_seen DESC
	`, kinds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Asset{}
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.ProgramID, &a.Kind, &a.Value); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *AssetStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID, limit, offset int) (Page[Asset], error) {
	limit, offset = ClampPageParams(limit, offset)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM assets WHERE organization_id=$1`, organizationID).Scan(&total); err != nil {
		return Page[Asset]{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, program_id, kind, value
		FROM assets
		WHERE organization_id=$1
		ORDER BY last_seen DESC
		LIMIT $2 OFFSET $3
	`, organizationID, limit, offset)
	if err != nil {
		return Page[Asset]{}, err
	}
	defer rows.Close()
	items := []Asset{}
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.OrganizationID, &a.ProgramID, &a.Kind, &a.Value); err != nil {
			return Page[Asset]{}, err
		}
		items = append(items, a)
	}
	return Page[Asset]{Items: items, Total: total, Limit: limit, Offset: offset}, rows.Err()
}

func (s *AssetStore) UpsertInOrganization(ctx context.Context, organizationID, programID uuid.UUID, kind, value string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO assets (organization_id, program_id, kind, value)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (organization_id, program_id, kind, value)
		DO UPDATE SET last_seen=now()
	`, organizationID, programID, kind, value)
	return err
}

func (s *AssetStore) ExistsInOrganization(ctx context.Context, organizationID, assetID uuid.UUID) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND organization_id=$2)`, assetID, organizationID).Scan(&ok)
	return ok, err
}
