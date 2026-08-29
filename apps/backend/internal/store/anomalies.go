package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RFAnomaly struct {
	ID             uuid.UUID `json:"id"`
	ReadingID      uuid.UUID `json:"reading_id"`
	AgentID        string    `json:"agent_id"`
	Metric         string    `json:"metric"`
	Value          float64   `json:"value"`
	BaselineMean   float64   `json:"baseline_mean"`
	BaselineStddev float64   `json:"baseline_stddev"`
	ZScore         float64   `json:"z_score"`
	DetectedAt     time.Time `json:"detected_at"`
}

type AnomalyStore struct {
	pool *pgxpool.Pool
}

func NewAnomalyStore(pool *pgxpool.Pool) *AnomalyStore {
	return &AnomalyStore{pool: pool}
}

func (s *AnomalyStore) Insert(ctx context.Context, a RFAnomaly) error {
	_, err := s.InsertReturning(ctx, a)
	return err
}

func (s *AnomalyStore) InsertReturning(ctx context.Context, a RFAnomaly) (RFAnomaly, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO rf_anomalies (reading_id, agent_id, metric, value, baseline_mean, baseline_stddev, z_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, detected_at
	`, a.ReadingID, a.AgentID, a.Metric, a.Value, a.BaselineMean, a.BaselineStddev, a.ZScore).Scan(&a.ID, &a.DetectedAt)
	return a, err
}

func (s *AnomalyStore) Recent(ctx context.Context, limit int) ([]RFAnomaly, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, reading_id, agent_id, metric, value, baseline_mean, baseline_stddev, z_score, detected_at
		FROM rf_anomalies
		ORDER BY detected_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RFAnomaly
	for rows.Next() {
		var a RFAnomaly
		if err := rows.Scan(&a.ID, &a.ReadingID, &a.AgentID, &a.Metric, &a.Value, &a.BaselineMean, &a.BaselineStddev, &a.ZScore, &a.DetectedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
