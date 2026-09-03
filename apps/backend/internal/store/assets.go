package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Asset struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ProgramID      uuid.UUID `json:"program_id"`
	Kind           string    `json:"kind"`
	Value          string    `json:"value"`
	CPEURI         string    `json:"cpe_uri,omitempty"`
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
        SELECT id, organization_id, program_id, kind, value, cpe_uri FROM assets
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
		if err := rows.Scan(&a.ID, &a.OrganizationID, &a.ProgramID, &a.Kind, &a.Value, &a.CPEURI); err != nil {
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

func (s *AssetStore) UpsertWithCPE(ctx context.Context, programID uuid.UUID, kind, value, cpeURI string) error {
	return s.UpsertInOrganizationWithCPE(ctx, DefaultOrganizationID, programID, kind, value, cpeURI)
}

func (s *AssetStore) ListByKinds(ctx context.Context, kinds []string) ([]Asset, error) {
	if len(kinds) == 0 {
		return nil, nil
	}

	rows, err := s.pool.Query(ctx, `
        SELECT id, organization_id, program_id, kind, value, cpe_uri FROM assets
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
		if err := rows.Scan(&a.ID, &a.OrganizationID, &a.ProgramID, &a.Kind, &a.Value, &a.CPEURI); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *AssetStore) ListAlertCandidates(ctx context.Context, kinds []string) ([]Asset, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, program_id, kind, value, cpe_uri
		FROM assets
		WHERE cpe_uri <> '' OR kind = ANY($1)
		ORDER BY last_seen DESC
	`, kinds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Asset{}
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.OrganizationID, &a.ProgramID, &a.Kind, &a.Value, &a.CPEURI); err != nil {
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
        SELECT id, organization_id, program_id, kind, value, cpe_uri
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
		if err := rows.Scan(&a.ID, &a.OrganizationID, &a.ProgramID, &a.Kind, &a.Value, &a.CPEURI); err != nil {
			return Page[Asset]{}, err
		}
		items = append(items, a)
	}
	return Page[Asset]{Items: items, Total: total, Limit: limit, Offset: offset}, rows.Err()
}

func (s *AssetStore) UpsertInOrganization(ctx context.Context, organizationID, programID uuid.UUID, kind, value string) error {
	tag, err := s.pool.Exec(ctx, `
        INSERT INTO assets (organization_id, program_id, kind, value)
        SELECT $1, p.id, $3, $4
        FROM programs p
        WHERE p.id=$2 AND p.organization_id=$1
        ON CONFLICT (organization_id, program_id, kind, value)
        DO UPDATE SET last_seen=now()
    `, organizationID, programID, kind, value)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *AssetStore) UpsertInOrganizationWithCPE(ctx context.Context, organizationID, programID uuid.UUID, kind, value, cpeURI string) error {
	tag, err := s.pool.Exec(ctx, `
        INSERT INTO assets (organization_id, program_id, kind, value, cpe_uri)
        SELECT $1, p.id, $3, $4, $5
        FROM programs p
        WHERE p.id=$2 AND p.organization_id=$1
        ON CONFLICT (organization_id, program_id, kind, value)
        DO UPDATE SET last_seen=now(), cpe_uri=EXCLUDED.cpe_uri
    `, organizationID, programID, kind, value, cpeURI)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *AssetStore) UpdateInOrganization(ctx context.Context, organizationID, assetID, programID uuid.UUID, kind, value string) (Asset, error) {
	var asset Asset
	err := s.pool.QueryRow(ctx, `
        UPDATE assets a
        SET program_id=p.id, kind=$4, value=$5, last_seen=now()
        FROM programs p
        WHERE a.id=$2
          AND a.organization_id=$1
          AND p.id=$3
          AND p.organization_id=$1
        RETURNING a.id, a.organization_id, a.program_id, a.kind, a.value, a.cpe_uri
    `, organizationID, assetID, programID, kind, value).Scan(
		&asset.ID, &asset.OrganizationID, &asset.ProgramID, &asset.Kind, &asset.Value, &asset.CPEURI,
	)
	return asset, err
}

func (s *AssetStore) UpdateInOrganizationWithCPE(ctx context.Context, organizationID, assetID, programID uuid.UUID, kind, value, cpeURI string) (Asset, error) {
	var asset Asset
	err := s.pool.QueryRow(ctx, `
        UPDATE assets a
        SET program_id=p.id, kind=$4, value=$5, cpe_uri=$6, last_seen=now()
        FROM programs p
        WHERE a.id=$2
          AND a.organization_id=$1
          AND p.id=$3
          AND p.organization_id=$1
        RETURNING a.id, a.organization_id, a.program_id, a.kind, a.value, a.cpe_uri
    `, organizationID, assetID, programID, kind, value, cpeURI).Scan(
		&asset.ID, &asset.OrganizationID, &asset.ProgramID, &asset.Kind, &asset.Value, &asset.CPEURI,
	)
	return asset, err
}

func (s *AssetStore) DeleteInOrganization(ctx context.Context, organizationID, assetID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM assets WHERE id=$1 AND organization_id=$2`, assetID, organizationID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *AssetStore) ExistsInOrganization(ctx context.Context, organizationID, assetID uuid.UUID) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND organization_id=$2)`, assetID, organizationID).Scan(&ok)
	return ok, err
}
