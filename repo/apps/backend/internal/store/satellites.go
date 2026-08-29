package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Satellite struct {
	ID        uuid.UUID `json:"id"`
	NoradID   int       `json:"norad_id"`
	Name      string    `json:"name"`
	TLELine1  string    `json:"tle_line1"`
	TLELine2  string    `json:"tle_line2"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SatelliteStore struct {
	pool *pgxpool.Pool
}

func NewSatelliteStore(pool *pgxpool.Pool) *SatelliteStore {
	return &SatelliteStore{pool: pool}
}

func (s *SatelliteStore) Upsert(ctx context.Context, noradID int, name, line1, line2 string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO satellites (norad_id, name, tle_line1, tle_line2)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (norad_id)
		DO UPDATE SET name = $2, tle_line1 = $3, tle_line2 = $4, updated_at = now()
	`, noradID, name, line1, line2)
	return err
}

func (s *SatelliteStore) List(ctx context.Context) ([]Satellite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, norad_id, name, tle_line1, tle_line2, updated_at
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
		if err := rows.Scan(&sat.ID, &sat.NoradID, &sat.Name, &sat.TLELine1, &sat.TLELine2, &sat.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, sat)
	}
	return out, rows.Err()
}
