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

func (s *SpaceCVEStore) List(ctx context.Context, limit, offset int) (Page[SpaceCVE], error) {
	limit, offset = ClampPageParams(limit, offset)

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM space_cves`).Scan(&total); err != nil {
		return Page[SpaceCVE]{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT cve_id, description, published, severity, score, keyword, synced_at
		FROM space_cves
		ORDER BY published DESC NULLS LAST
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return Page[SpaceCVE]{}, err
	}
	defer rows.Close()

	items := []SpaceCVE{}
	for rows.Next() {
		var c SpaceCVE
		var published *time.Time
		var score *float64
		if err := rows.Scan(&c.CVEID, &c.Description, &published, &c.Severity, &score, &c.Keyword, &c.SyncedAt); err != nil {
			return Page[SpaceCVE]{}, err
		}
		if published != nil {
			c.Published = *published
		}
		if score != nil {
			c.Score = *score
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return Page[SpaceCVE]{}, err
	}

	return Page[SpaceCVE]{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}
