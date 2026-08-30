package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
)

var ErrDetectionRuleNotFound = errors.New("store: detection rule not found")

const MaxEnabledDetectionRules = 200

type DetectionMatch struct {
	ID         uuid.UUID  `json:"id"`
	RuleID     uuid.UUID  `json:"rule_id"`
	AnomalyID  uuid.UUID  `json:"anomaly_id"`
	IncidentID *uuid.UUID `json:"incident_id,omitempty"`
	AgentID    string     `json:"agent_id"`
	Metric     string     `json:"metric"`
	ZScore     float64    `json:"z_score"`
	MatchedAt  time.Time  `json:"matched_at"`
}

type DetectionRuleStore struct {
	pool *pgxpool.Pool
}

func NewDetectionRuleStore(pool *pgxpool.Pool) *DetectionRuleStore {
	return &DetectionRuleStore{pool: pool}
}

func (s *DetectionRuleStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID, limit, offset int) (Page[detection.Rule], error) {
	limit, offset = ClampPageParams(limit, offset)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM rf_detection_rules WHERE organization_id=$1`, organizationID).Scan(&total); err != nil {
		return Page[detection.Rule]{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, name, description, enabled, metric, operator, threshold, severity,
		       cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
		FROM rf_detection_rules
		WHERE organization_id = $1
		ORDER BY enabled DESC, updated_at DESC, name ASC
		LIMIT $2 OFFSET $3
	`, organizationID, limit, offset)
	if err != nil {
		return Page[detection.Rule]{}, err
	}
	defer rows.Close()
	items := []detection.Rule{}
	for rows.Next() {
		rule, err := scanDetectionRule(rows)
		if err != nil {
			return Page[detection.Rule]{}, err
		}
		items = append(items, rule)
	}
	if err := rows.Err(); err != nil {
		return Page[detection.Rule]{}, err
	}
	return Page[detection.Rule]{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *DetectionRuleStore) ListEnabledByOrganization(ctx context.Context, organizationID uuid.UUID) ([]detection.Rule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, name, description, enabled, metric, operator, threshold, severity,
		       cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
		FROM rf_detection_rules
		WHERE organization_id = $1 AND enabled = true
		ORDER BY threshold DESC, name ASC
		LIMIT $2
	`, organizationID, MaxEnabledDetectionRules)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []detection.Rule{}
	for rows.Next() {
		rule, err := scanDetectionRule(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, rule)
	}
	return items, rows.Err()
}

func (s *DetectionRuleStore) EnabledCountByOrganization(ctx context.Context, organizationID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM rf_detection_rules WHERE organization_id=$1 AND enabled = true`, organizationID).Scan(&count)
	return count, err
}

func (s *DetectionRuleStore) GetByOrganization(ctx context.Context, organizationID, id uuid.UUID) (detection.Rule, error) {
	rule, err := scanDetectionRule(s.pool.QueryRow(ctx, `
		SELECT id, organization_id, name, description, enabled, metric, operator, threshold, severity,
		       cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
		FROM rf_detection_rules
		WHERE organization_id = $1 AND id = $2
	`, organizationID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return detection.Rule{}, ErrDetectionRuleNotFound
	}
	return rule, err
}

func (s *DetectionRuleStore) CreateInOrganization(ctx context.Context, organizationID uuid.UUID, rule detection.Rule, actorID uuid.UUID) (detection.Rule, error) {
	return scanDetectionRule(s.pool.QueryRow(ctx, `
		INSERT INTO rf_detection_rules (
			organization_id, name, description, enabled, metric, operator, threshold, severity,
			cooldown_seconds, auto_incident, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		RETURNING id, organization_id, name, description, enabled, metric, operator, threshold, severity,
		          cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
	`, organizationID, rule.Name, rule.Description, rule.Enabled, rule.Metric, rule.Operator, rule.Threshold,
		rule.Severity, rule.CooldownSeconds, rule.AutoIncident, actorID))
}

func (s *DetectionRuleStore) UpdateInOrganization(ctx context.Context, organizationID uuid.UUID, rule detection.Rule, actorID uuid.UUID) (detection.Rule, error) {
	updated, err := scanDetectionRule(s.pool.QueryRow(ctx, `
		UPDATE rf_detection_rules
		SET name = $3,
		    description = $4,
		    enabled = $5,
		    metric = $6,
		    operator = $7,
		    threshold = $8,
		    severity = $9,
		    cooldown_seconds = $10,
		    auto_incident = $11,
		    updated_by = $12,
		    updated_at = now()
		WHERE organization_id = $1 AND id = $2
		RETURNING id, organization_id, name, description, enabled, metric, operator, threshold, severity,
		          cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
	`, organizationID, rule.ID, rule.Name, rule.Description, rule.Enabled, rule.Metric, rule.Operator, rule.Threshold,
		rule.Severity, rule.CooldownSeconds, rule.AutoIncident, actorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return detection.Rule{}, ErrDetectionRuleNotFound
	}
	return updated, err
}

func (s *DetectionRuleStore) RecordMatchIfAllowed(ctx context.Context, organizationID uuid.UUID, rule detection.Rule, anomalyID uuid.UUID, agentID, metric string, zScore float64) (DetectionMatch, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DetectionMatch{}, false, err
	}
	defer tx.Rollback(ctx)

	lockKey := organizationID.String() + ":" + rule.ID.String() + ":" + agentID
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return DetectionMatch{}, false, err
	}
	if rule.CooldownSeconds > 0 {
		var recent bool
		since := time.Now().UTC().Add(-time.Duration(rule.CooldownSeconds) * time.Second)
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM rf_detection_matches
				WHERE organization_id = $1 AND rule_id = $2 AND agent_id = $3 AND matched_at >= $4
			)
		`, organizationID, rule.ID, agentID, since).Scan(&recent); err != nil {
			return DetectionMatch{}, false, err
		}
		if recent {
			return DetectionMatch{}, false, nil
		}
	}

	var match DetectionMatch
	err = tx.QueryRow(ctx, `
		INSERT INTO rf_detection_matches (organization_id, rule_id, anomaly_id, agent_id, metric, z_score)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (rule_id, anomaly_id) DO NOTHING
		RETURNING id, rule_id, anomaly_id, incident_id, agent_id, metric, z_score, matched_at
	`, organizationID, rule.ID, anomalyID, agentID, metric, zScore).Scan(
		&match.ID, &match.RuleID, &match.AnomalyID, &match.IncidentID, &match.AgentID,
		&match.Metric, &match.ZScore, &match.MatchedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DetectionMatch{}, false, nil
	}
	if err != nil {
		return DetectionMatch{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DetectionMatch{}, false, err
	}
	return match, true, nil
}

func (s *DetectionRuleStore) AttachIncidentInOrganization(ctx context.Context, organizationID uuid.UUID, matchIDs []uuid.UUID, incidentID uuid.UUID) error {
	if len(matchIDs) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE rf_detection_matches
		SET incident_id = $3
		WHERE organization_id = $1 AND id = ANY($2::uuid[])
	`, organizationID, matchIDs, incidentID)
	return err
}

type detectionRuleScanner interface {
	Scan(dest ...interface{}) error
}

func scanDetectionRule(row detectionRuleScanner) (detection.Rule, error) {
	var rule detection.Rule
	err := row.Scan(
		&rule.ID, &rule.OrganizationID, &rule.Name, &rule.Description, &rule.Enabled, &rule.Metric, &rule.Operator,
		&rule.Threshold, &rule.Severity, &rule.CooldownSeconds, &rule.AutoIncident,
		&rule.CreatedBy, &rule.UpdatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
}
