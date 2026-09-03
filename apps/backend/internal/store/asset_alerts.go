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
	KEV                 bool       `json:"kev"`
	KEVDueDate          *time.Time `json:"kev_due_date,omitempty"`
	EPSSScore           *float64   `json:"epss_score,omitempty"`
	EPSSPercentile      *float64   `json:"epss_percentile,omitempty"`
	MatchedKeyword      string     `json:"matched_keyword"`
	ApplicabilityStatus string     `json:"applicability_status"`
	MatchMethod         string     `json:"match_method"`
	Confidence          int        `json:"confidence"`
	ContextOnly         bool       `json:"context_only"`
	Severity            string     `json:"severity"`
	LifecycleStatus     string     `json:"lifecycle_status"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	LastEvaluatedAt     time.Time  `json:"last_evaluated_at"`
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
		INSERT INTO asset_cve_alerts (asset_id, cve_id, matched_keyword, severity, applicability_status, match_method, confidence, context_only, lifecycle_status, last_evaluated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'open', now())
		ON CONFLICT (asset_id, cve_id) DO UPDATE SET
		  matched_keyword=EXCLUDED.matched_keyword, severity=EXCLUDED.severity, applicability_status=EXCLUDED.applicability_status,
		  match_method=EXCLUDED.match_method, confidence=EXCLUDED.confidence, context_only=EXCLUDED.context_only,
		  lifecycle_status=CASE WHEN asset_cve_alerts.lifecycle_status='suppressed' THEN 'suppressed' ELSE 'open' END,
		  resolved_at=NULL, last_evaluated_at=now()
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
			a.cve_id, c.description, COALESCE(c.score, 0), c.kev, c.kev_due_date, c.epss_score, c.epss_percentile,
			a.matched_keyword, a.severity, a.applicability_status, a.match_method, a.confidence, a.context_only,
			a.lifecycle_status, a.resolved_at, a.last_evaluated_at, a.acknowledged, a.acknowledged_by, a.acknowledged_at, a.created_at
		FROM asset_cve_alerts a
		JOIN assets ast ON ast.id = a.asset_id
		JOIN space_cves c ON c.cve_id = a.cve_id
		ORDER BY CASE a.lifecycle_status WHEN 'open' THEN 0 WHEN 'reviewed' THEN 1 WHEN 'resolved' THEN 2 ELSE 3 END, c.kev DESC, c.epss_score DESC NULLS LAST, a.created_at DESC
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
			&al.CVEID, &al.CVEDescription, &al.CVEScore, &al.KEV, &al.KEVDueDate, &al.EPSSScore, &al.EPSSPercentile,
			&al.MatchedKeyword, &al.Severity, &al.ApplicabilityStatus, &al.MatchMethod, &al.Confidence, &al.ContextOnly,
			&al.LifecycleStatus, &al.ResolvedAt, &al.LastEvaluatedAt, &al.Acknowledged, &al.AcknowledgedBy, &al.AcknowledgedAt, &al.CreatedAt,
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
		SET acknowledged = true, acknowledged_by = $2, acknowledged_at = now(), lifecycle_status = CASE WHEN lifecycle_status='open' THEN 'reviewed' ELSE lifecycle_status END, last_evaluated_at=now()
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
		SELECT a.id, a.asset_id, ast.kind, ast.value, a.cve_id, c.description, COALESCE(c.score, 0), c.kev, c.kev_due_date, c.epss_score, c.epss_percentile,
		       a.matched_keyword, a.severity, a.applicability_status, a.match_method, a.confidence, a.context_only,
		       a.lifecycle_status, a.resolved_at, a.last_evaluated_at, a.acknowledged, a.acknowledged_by, a.acknowledged_at, a.created_at
		FROM asset_cve_alerts a
		JOIN assets ast ON ast.id = a.asset_id
		JOIN space_cves c ON c.cve_id = a.cve_id
		WHERE a.lifecycle_status='open' AND a.created_at >= $1
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
			&al.CVEScore, &al.KEV, &al.KEVDueDate, &al.EPSSScore, &al.EPSSPercentile, &al.MatchedKeyword, &al.Severity, &al.ApplicabilityStatus, &al.MatchMethod, &al.Confidence, &al.ContextOnly, &al.LifecycleStatus, &al.ResolvedAt, &al.LastEvaluatedAt, &al.Acknowledged, &al.AcknowledgedBy,
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
		SELECT a.id,a.asset_id,ast.kind,ast.value,a.cve_id,c.description,COALESCE(c.score,0),c.kev,c.kev_due_date,c.epss_score,c.epss_percentile,
		       a.matched_keyword,a.severity,a.applicability_status,a.match_method,a.confidence,a.context_only,
		       a.lifecycle_status,a.resolved_at,a.last_evaluated_at,a.acknowledged,a.acknowledged_by,a.acknowledged_at,a.created_at
		FROM asset_cve_alerts a
		JOIN assets ast ON ast.id=a.asset_id AND ast.organization_id=$1
		JOIN space_cves c ON c.cve_id=a.cve_id
		ORDER BY CASE a.lifecycle_status WHEN 'open' THEN 0 WHEN 'reviewed' THEN 1 WHEN 'resolved' THEN 2 ELSE 3 END,c.kev DESC,c.epss_score DESC NULLS LAST,a.created_at DESC LIMIT $2 OFFSET $3
	`, organizationID, limit, offset)
	if err != nil {
		return Page[AssetCVEAlert]{}, err
	}
	defer rows.Close()
	items := []AssetCVEAlert{}
	for rows.Next() {
		var al AssetCVEAlert
		if err := rows.Scan(&al.ID, &al.AssetID, &al.AssetKind, &al.AssetValue, &al.CVEID, &al.CVEDescription, &al.CVEScore, &al.KEV, &al.KEVDueDate, &al.EPSSScore, &al.EPSSPercentile, &al.MatchedKeyword, &al.Severity, &al.ApplicabilityStatus, &al.MatchMethod, &al.Confidence, &al.ContextOnly, &al.LifecycleStatus, &al.ResolvedAt, &al.LastEvaluatedAt, &al.Acknowledged, &al.AcknowledgedBy, &al.AcknowledgedAt, &al.CreatedAt); err != nil {
			return Page[AssetCVEAlert]{}, err
		}
		items = append(items, al)
	}
	return Page[AssetCVEAlert]{Items: items, Total: total, Limit: limit, Offset: offset}, rows.Err()
}

func (s *AssetAlertStore) AcknowledgeInOrganization(ctx context.Context, organizationID, alertID, userID uuid.UUID) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE asset_cve_alerts a SET acknowledged=true,acknowledged_by=$3,acknowledged_at=now(),lifecycle_status=CASE WHEN a.lifecycle_status='open' THEN 'reviewed' ELSE a.lifecycle_status END,last_evaluated_at=now()
		FROM assets ast WHERE a.id=$1 AND ast.id=a.asset_id AND ast.organization_id=$2
	`, alertID, organizationID, userID)
	return tag.RowsAffected() > 0, err
}

func (s *AssetAlertStore) ResolveCVEExcept(ctx context.Context, cveID string, activeAssetIDs []uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE asset_cve_alerts
		SET lifecycle_status='resolved', resolved_at=now(), last_evaluated_at=now()
		WHERE cve_id=$1 AND lifecycle_status IN ('open','reviewed') AND NOT (asset_id = ANY($2::uuid[]))
	`, cveID, activeAssetIDs)
	return err
}
