package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetCVEAlert struct {
	ID             uuid.UUID  `json:"id"`
	AssetID        uuid.UUID  `json:"asset_id"`
	AssetKind      string     `json:"asset_kind"`
	AssetValue     string     `json:"asset_value"`
	CVEID          string     `json:"cve_id"`
	CVEDescription string     `json:"cve_description"`
	CVEScore       float64    `json:"cve_score"`
	MatchedKeyword string     `json:"matched_keyword"`
	Severity       string     `json:"severity"`
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedBy *uuid.UUID `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AssetAlertStore struct {
	pool *pgxpool.Pool
}

func NewAssetAlertStore(pool *pgxpool.Pool) *AssetAlertStore {
	return &AssetAlertStore{pool: pool}
}

func (s *AssetAlertStore) Create(ctx context.Context, assetID uuid.UUID, cveID, matchedKeyword, severity string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO asset_cve_alerts (asset_id, cve_id, matched_keyword, severity)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (asset_id, cve_id) DO NOTHING
	`, assetID, cveID, matchedKeyword, severity)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *AssetAlertStore) List(ctx context.Context, limit, offset int) (Page[AssetCVEAlert], error) {
	limit, offset = ClampPageParams(limit, offset)

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM asset_cve_alerts`).Scan(&total); err != nil {
		return Page[AssetCVEAlert]{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT
			a.id, a.asset_id, ast.kind, ast.value,
			a.cve_id, c.description, COALESCE(c.score, 0),
			a.matched_keyword, a.severity,
			a.acknowledged, a.acknowledged_by, a.acknowledged_at, a.created_at
		FROM asset_cve_alerts a
		JOIN assets ast ON ast.id = a.asset_id
		JOIN space_cves c ON c.cve_id = a.cve_id
		ORDER BY a.acknowledged ASC, a.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return Page[AssetCVEAlert]{}, err
	}
	defer rows.Close()

	items := []AssetCVEAlert{}
	for rows.Next() {
		var al AssetCVEAlert
		if err := rows.Scan(
			&al.ID, &al.AssetID, &al.AssetKind, &al.AssetValue,
			&al.CVEID, &al.CVEDescription, &al.CVEScore,
			&al.MatchedKeyword, &al.Severity,
			&al.Acknowledged, &al.AcknowledgedBy, &al.AcknowledgedAt, &al.CreatedAt,
		); err != nil {
			return Page[AssetCVEAlert]{}, err
		}
		items = append(items, al)
	}
	if err := rows.Err(); err != nil {
		return Page[AssetCVEAlert]{}, err
	}

	return Page[AssetCVEAlert]{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *AssetAlertStore) Acknowledge(ctx context.Context, alertID, userID uuid.UUID) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE asset_cve_alerts
		SET acknowledged = true, acknowledged_by = $2, acknowledged_at = now()
		WHERE id = $1
	`, alertID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
