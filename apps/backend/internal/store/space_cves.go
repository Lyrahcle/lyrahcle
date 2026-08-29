package store

import (
	"context"
	"fmt"
	"strings"
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

type SpaceCVEFilter struct {
	Severity      string
	Keyword       string
	PublishedFrom time.Time
	PublishedTo   time.Time
	SortBy        string
	SortDesc      bool
}

var validSpaceCVESortFields = map[string]string{
	"published": "published",
	"severity":  "severity_rank",
	"score":     "score",
	"cve_id":    "cve_id",
}

func (s *SpaceCVEStore) List(ctx context.Context, limit, offset int, filter SpaceCVEFilter) (Page[SpaceCVE], error) {
	limit, offset = ClampPageParams(limit, offset)

	where := []string{"1=1"}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.Severity != "" {
		where = append(where, "lower(severity) = lower("+arg(filter.Severity)+")")
	}
	if filter.Keyword != "" {
		where = append(where, "lower(keyword) = lower("+arg(filter.Keyword)+")")
	}
	if !filter.PublishedFrom.IsZero() {
		where = append(where, "published >= "+arg(filter.PublishedFrom))
	}
	if !filter.PublishedTo.IsZero() {
		where = append(where, "published <= "+arg(filter.PublishedTo))
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	countSQL := "SELECT count(*) FROM space_cves WHERE " + whereClause
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[SpaceCVE]{}, err
	}

	sortCol, ok := validSpaceCVESortFields[filter.SortBy]
	if !ok {
		sortCol = "published"
	}
	dir := "ASC"
	if filter.SortDesc {
		dir = "DESC"
	}

	orderExpr := sortCol
	if sortCol == "severity_rank" {
		orderExpr = `CASE lower(severity)
			WHEN 'critical' THEN 4
			WHEN 'high' THEN 3
			WHEN 'medium' THEN 2
			WHEN 'low' THEN 1
			ELSE 0
		END`
	}

	querySQL := fmt.Sprintf(`
		SELECT cve_id, description, published, severity, score, keyword, synced_at
		FROM space_cves
		WHERE %s
		ORDER BY %s %s NULLS LAST, cve_id ASC
		LIMIT %s OFFSET %s
	`, whereClause, orderExpr, dir, arg(limit), arg(offset))

	rows, err := s.pool.Query(ctx, querySQL, args...)
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

func (s *SpaceCVEStore) DistinctKeywords(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT keyword FROM space_cves ORDER BY keyword ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *SpaceCVEStore) GetByID(ctx context.Context, cveID string) (SpaceCVE, error) {
	var c SpaceCVE
	var published *time.Time
	var score *float64

	err := s.pool.QueryRow(ctx, `
		SELECT cve_id, description, published, severity, score, keyword, synced_at
		FROM space_cves WHERE cve_id = $1
	`, cveID).Scan(&c.CVEID, &c.Description, &published, &c.Severity, &score, &c.Keyword, &c.SyncedAt)
	if err != nil {
		return SpaceCVE{}, err
	}
	if published != nil {
		c.Published = *published
	}
	if score != nil {
		c.Score = *score
	}
	return c, nil
}
