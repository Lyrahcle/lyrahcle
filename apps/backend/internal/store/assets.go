package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Asset struct {
	ID        uuid.UUID `json:"id"`
	ProgramID uuid.UUID `json:"program_id"`
	Kind      string    `json:"kind"`
	Value     string    `json:"value"`
}

type AssetStore struct {
	pool *pgxpool.Pool
}

func NewAssetStore(pool *pgxpool.Pool) *AssetStore {
	return &AssetStore{pool: pool}
}

func (s *AssetStore) List(ctx context.Context) ([]Asset, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, program_id, kind, value FROM assets ORDER BY last_seen DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.ProgramID, &a.Kind, &a.Value); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *AssetStore) Upsert(ctx context.Context, programID uuid.UUID, kind, value string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO assets (program_id, kind, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (program_id, kind, value)
		DO UPDATE SET last_seen = now()
	`, programID, kind, value)
	return err
}
