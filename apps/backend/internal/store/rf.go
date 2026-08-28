package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RFReading struct {
	ID          uuid.UUID `json:"id"`
	AgentID     string    `json:"agent_id"`
	FrequencyHz int64     `json:"frequency_hz"`
	PowerDbm    float64   `json:"power_dbm"`
	CapturedAt  time.Time `json:"captured_at"`
}

type RFStore struct {
	pool *pgxpool.Pool
}

func NewRFStore(pool *pgxpool.Pool) *RFStore {
	return &RFStore{pool: pool}
}

func (s *RFStore) Insert(ctx context.Context, r *RFReading) error {
	return s.pool.QueryRow(ctx, `
		INSERT INTO rf_readings (agent_id, frequency_hz, power_dbm, captured_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, r.AgentID, r.FrequencyHz, r.PowerDbm, r.CapturedAt).Scan(&r.ID)
}

func (s *RFStore) Recent(ctx context.Context, limit int) ([]RFReading, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_id, frequency_hz, power_dbm, captured_at
		FROM rf_readings
		ORDER BY captured_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RFReading
	for rows.Next() {
		var r RFReading
		if err := rows.Scan(&r.ID, &r.AgentID, &r.FrequencyHz, &r.PowerDbm, &r.CapturedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *RFStore) RecentForAgent(ctx context.Context, agentID string, limit int) ([]RFReading, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_id, frequency_hz, power_dbm, captured_at
		FROM rf_readings
		WHERE agent_id = $1
		ORDER BY captured_at DESC
		LIMIT $2
	`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RFReading
	for rows.Next() {
		var r RFReading
		if err := rows.Scan(&r.ID, &r.AgentID, &r.FrequencyHz, &r.PowerDbm, &r.CapturedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
