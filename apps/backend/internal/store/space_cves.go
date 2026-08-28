package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SpaceCVE struct {
	CVEID       string    `json:"cve_id"`
	Description string    `json:"description"`
	Published   time.Time `json:"published"`
	Severity    string    `json:"severity"`
	Score       float64   `json:"score"`
	Keyword     string    `json:"keyword"`
	SyncedAt    time.Time `json:"synced_at"`
}

type SpaceCVEStore struct {
	pool *pgxpool.Pool
}

func NewSpaceCVEStore(pool *pgxpool.Pool) *SpaceCVEStore {
	return &SpaceCVEStore{pool: pool}
}

func (s *SpaceCVEStore) Upsert(ctx context.Context, cveID, description string, published time.Time, severity string, score float64, keyword string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO space_cves (cve_id, description, published, severity, score, keyword)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cve_id)
		DO UPDATE SET description = $2, published = $3, severity = $4, score = $5, keyword = $6, synced_at = now()
	`, cveID, description, published, severity, score, keyword)
	return err
}

func (s *SpaceCVEStore) List(ctx context.Context, limit int) ([]SpaceCVE, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cve_id, description, published, severity, score, keyword, synced_at
		FROM space_cves
		ORDER BY published DESC NULLS LAST
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SpaceCVE
	for rows.Next() {
		var c SpaceCVE
		var published *time.Time
		var score *float64
		if err := rows.Scan(&c.CVEID, &c.Description, &published, &c.Severity, &score, &c.Keyword, &c.SyncedAt); err != nil {
			return nil, err
		}
		if published != nil {
			c.Published = *published
		}
		if score != nil {
			c.Score = *score
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
