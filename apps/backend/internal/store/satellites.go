package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/tlequality"
)

type Satellite struct {
	ID        uuid.UUID  `json:"id"`
	NoradID   int        `json:"norad_id"`
	Name      string     `json:"name"`
	TLELine1  string     `json:"tle_line1"`
	TLELine2  string     `json:"tle_line2"`
	TLEEpoch  *time.Time `json:"tle_epoch,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SatelliteStore struct {
	pool *pgxpool.Pool
}

func NewSatelliteStore(pool *pgxpool.Pool) *SatelliteStore {
	return &SatelliteStore{pool: pool}
}

func (s *SatelliteStore) Upsert(ctx context.Context, noradID int, name, line1, line2 string) error {
	epoch, err := tlequality.Epoch(line1)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO satellites (norad_id, name, tle_line1, tle_line2, tle_epoch)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (norad_id)
		DO UPDATE SET name = $2, tle_line1 = $3, tle_line2 = $4, tle_epoch = $5, updated_at = now()
	`, noradID, name, line1, line2, epoch)
	return err
}

func (s *SatelliteStore) Update(ctx context.Context, id uuid.UUID, noradID int, name, line1, line2 string) (Satellite, error) {
	epoch, err := tlequality.Epoch(line1)
	if err != nil {
		return Satellite{}, err
	}
	var sat Satellite
	err = s.pool.QueryRow(ctx, `
		UPDATE satellites
		SET norad_id = $2, name = $3, tle_line1 = $4, tle_line2 = $5, tle_epoch = $6, updated_at = now()
		WHERE id = $1
		RETURNING id, norad_id, name, tle_line1, tle_line2, tle_epoch, updated_at
	`, id, noradID, name, line1, line2, epoch).Scan(
		&sat.ID, &sat.NoradID, &sat.Name, &sat.TLELine1, &sat.TLELine2, &sat.TLEEpoch, &sat.UpdatedAt,
	)
	return sat, err
}

func (s *SatelliteStore) List(ctx context.Context) ([]Satellite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, norad_id, name, tle_line1, tle_line2, tle_epoch, updated_at
		FROM satellites
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Satellite
	for rows.Next() {
		var sat Satellite
		if err := rows.Scan(&sat.ID, &sat.NoradID, &sat.Name, &sat.TLELine1, &sat.TLELine2, &sat.TLEEpoch, &sat.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, sat)
	}
	return out, rows.Err()
}

func (s *SatelliteStore) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM satellites WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func (s *SatelliteStore) Get(ctx context.Context, id uuid.UUID) (Satellite, error) {
	var sat Satellite
	err := s.pool.QueryRow(ctx, `
		SELECT id, norad_id, name, tle_line1, tle_line2, tle_epoch, updated_at
		FROM satellites
		WHERE id = $1
	`, id).Scan(&sat.ID, &sat.NoradID, &sat.Name, &sat.TLELine1, &sat.TLELine2, &sat.TLEEpoch, &sat.UpdatedAt)
	return sat, err
}
