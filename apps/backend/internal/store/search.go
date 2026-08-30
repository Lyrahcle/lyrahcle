package store

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SearchResult struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Label    string `json:"label"`
	Subtitle string `json:"subtitle"`
	Tab      string `json:"tab"`
}

type SearchStore struct {
	pool *pgxpool.Pool
}

func NewSearchStore(pool *pgxpool.Pool) *SearchStore {
	return &SearchStore{pool: pool}
}

func (s *SearchStore) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if limit < 1 || limit > 20 {
		limit = 12
	}
	rows, err := s.pool.Query(ctx, `
        SELECT type, id, label, subtitle, tab
        FROM (
            SELECT 'satellites'::text AS type, id::text AS id, name AS label, 'NORAD ' || norad_id::text AS subtitle, 'orbital'::text AS tab, 1 AS rank
            FROM satellites WHERE lower(name) LIKE '%' || lower($1) || '%' OR norad_id::text = $1
            UNION ALL
            SELECT 'assets', id::text, value, kind, 'assets', 2
            FROM assets WHERE lower(value) LIKE '%' || lower($1) || '%' OR lower(kind) LIKE '%' || lower($1) || '%'
            UNION ALL
            SELECT 'cves', cve_id, cve_id, coalesce(severity, 'unknown'), 'intel', 3
            FROM space_cves WHERE lower(cve_id) LIKE '%' || lower($1) || '%' OR lower(description) LIKE '%' || lower($1) || '%'
            UNION ALL
            SELECT 'incidents', id::text, reference, title, 'incidents', 4
            FROM incidents WHERE lower(reference) LIKE '%' || lower($1) || '%' OR lower(title) LIKE '%' || lower($1) || '%'
        ) results
        ORDER BY rank, lower(label)
        LIMIT $2
    `, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SearchResult, 0)
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.Type, &item.ID, &item.Label, &item.Subtitle, &item.Tab); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *SearchStore) SearchByOrganization(ctx context.Context, organizationID uuid.UUID, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if limit < 1 || limit > 20 {
		limit = 12
	}
	rows, err := s.pool.Query(ctx, `
        SELECT type, id, label, subtitle, tab FROM (
            SELECT 'satellites'::text, id::text, name, 'NORAD ' || norad_id::text, 'orbital'::text, 1 AS rank
            FROM satellites WHERE lower(name) LIKE '%' || lower($1) || '%' OR norad_id::text=$1
            UNION ALL
            SELECT 'assets', id::text, value, kind, 'assets', 2
            FROM assets WHERE organization_id=$2 AND (lower(value) LIKE '%' || lower($1) || '%' OR lower(kind) LIKE '%' || lower($1) || '%')
            UNION ALL
            SELECT 'cves', cve_id, cve_id, coalesce(severity,'unknown'), 'intel', 3
            FROM space_cves WHERE lower(cve_id) LIKE '%' || lower($1) || '%' OR lower(description) LIKE '%' || lower($1) || '%'
            UNION ALL
            SELECT 'incidents', id::text, reference, title, 'incidents', 4
            FROM incidents WHERE organization_id=$2 AND (lower(reference) LIKE '%' || lower($1) || '%' OR lower(title) LIKE '%' || lower($1) || '%')
        ) results(type,id,label,subtitle,tab,rank)
        ORDER BY rank, lower(label) LIMIT $3
    `, query, organizationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SearchResult{}
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.Type, &item.ID, &item.Label, &item.Subtitle, &item.Tab); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
