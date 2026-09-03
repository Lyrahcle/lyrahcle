package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetCVEAlert struct {
	ID                  uuid.UUID  `json:"id"`
	AssetID             uuid.UUID  `json:"asset_id"`
	AssetKind           string     `json:"asset_kind"`
	AssetValue          string     `json:"asset_value"`
	CVEID               string     `json:"cve_id"`
	CVEDescription      string     `json:"cve_description"`
	CVEScore            float64    `json:"cve_score"`
	MatchedKeyword      string     `json:"matched_keyword"`
	ApplicabilityStatus string     `json:"applicability_status"`
	MatchMethod         string     `json:"match_method"`
	Confidence          int        `json:"confidence"`
	ContextOnly         bool       `json:"context_only"`
	Severity            string     `json:"severity"`
	Acknowledged        bool       `json:"acknowledged"`
	AcknowledgedBy      *uuid.UUID `json:"acknowledged_by,omitempty"`
	AcknowledgedAt      *time.Time `json:"acknowledged_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type AssetAlertStore struct {
	pool *pgxpool.Pool
}

func NewAssetAlertStore(pool *pgxpool.Pool) *AssetAlertStore {
	return &AssetAlertStore{pool: pool}
}

func (s *AssetAlertStore) Create(ctx context.Context, assetID uuid.UUID, cveID, matchedKeyword, severity string) (bool, error) {
	return s.CreateWithAssessment(ctx, assetID, cveID, matchedKeyword, severity, "unknown", "thematic_asset_kind", 20, true)
}

func (s *AssetAlertStore) CreateWithAssessment(ctx context.Context, assetID uuid.UUID, cveID, matchedKeyword, severity, applicabilityStatus, matchMethod string, confidence int, contextOnly bool) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO asset_cve_alerts (asset_id, cve_id, matched_keyword, severity, applicability_status, match_method, confidence, context_only)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (asset_id, cve_id) DO NOTHING
	`, assetID, cveID, matchedKeyword, severity, applicabilityStatus, matchMethod, confidence, contextOnly)
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
			a.matched_keyword, a.severity, a.applicability_status, a.match_method, a.confidence, a.context_only,
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
			&al.MatchedKeyword, &al.Severity, &al.ApplicabilityStatus, &al.MatchMethod, &al.Confidence, &al.ContextOnly,
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

func (s *AssetAlertStore) RecentUnacknowledged(ctx context.Context, since time.Time, limit int) ([]AssetCVEAlert, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.asset_id, ast.kind, ast.value, a.cve_id, c.description, COALESCE(c.score, 0),
		       a.matched_keyword, a.severity, a.applicability_status, a.match_method, a.confidence, a.context_only,
		       a.acknowledged, a.acknowledged_by, a.acknowledged_at, a.created_at
		FROM asset_cve_alerts a
		JOIN assets ast ON ast.id = a.asset_id
		JOIN space_cves c ON c.cve_id = a.cve_id
		WHERE a.acknowledged = false AND a.created_at >= $1
		ORDER BY a.created_at DESC
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AssetCVEAlert{}
	for rows.Next() {
		var al AssetCVEAlert
		if err := rows.Scan(&al.ID, &al.AssetID, &al.AssetKind, &al.AssetValue, &al.CVEID, &al.CVEDescription,
			&al.CVEScore, &al.MatchedKeyword, &al.Severity, &al.ApplicabilityStatus, &al.MatchMethod, &al.Confidence, &al.ContextOnly, &al.Acknowledged, &al.AcknowledgedBy,
			&al.AcknowledgedAt, &al.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, al)
	}
	return items, rows.Err()
}

func (s *AssetAlertStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID, limit, offset int) (Page[AssetCVEAlert], error) {
	limit, offset = ClampPageParams(limit, offset)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM asset_cve_alerts a JOIN assets ast ON ast.id=a.asset_id WHERE ast.organization_id=$1`, organizationID).Scan(&total); err != nil {
		return Page[AssetCVEAlert]{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.id,a.asset_id,ast.kind,ast.value,a.cve_id,c.description,COALESCE(c.score,0),
		       a.matched_keyword,a.severity,a.applicability_status,a.match_method,a.confidence,a.context_only,
		       a.acknowledged,a.acknowledged_by,a.acknowledged_at,a.created_at
		FROM asset_cve_alerts a
		JOIN assets ast ON ast.id=a.asset_id AND ast.organization_id=$1
		JOIN space_cves c ON c.cve_id=a.cve_id
		ORDER BY a.acknowledged ASC,a.created_at DESC LIMIT $2 OFFSET $3
	`, organizationID, limit, offset)
	if err != nil {
		return Page[AssetCVEAlert]{}, err
	}
	defer rows.Close()
	items := []AssetCVEAlert{}
	for rows.Next() {
		var al AssetCVEAlert
		if err := rows.Scan(&al.ID, &al.AssetID, &al.AssetKind, &al.AssetValue, &al.CVEID, &al.CVEDescription, &al.CVEScore, &al.MatchedKeyword, &al.Severity, &al.ApplicabilityStatus, &al.MatchMethod, &al.Confidence, &al.ContextOnly, &al.Acknowledged, &al.AcknowledgedBy, &al.AcknowledgedAt, &al.CreatedAt); err != nil {
			return Page[AssetCVEAlert]{}, err
		}
		items = append(items, al)
	}
	return Page[AssetCVEAlert]{Items: items, Total: total, Limit: limit, Offset: offset}, rows.Err()
}

func (s *AssetAlertStore) AcknowledgeInOrganization(ctx context.Context, organizationID, alertID, userID uuid.UUID) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE asset_cve_alerts a SET acknowledged=true,acknowledged_by=$3,acknowledged_at=now()
		FROM assets ast WHERE a.id=$1 AND ast.id=a.asset_id AND ast.organization_id=$2
	`, alertID, organizationID, userID)
	return tag.RowsAffected() > 0, err
}
