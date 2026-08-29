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

func (s *DetectionRuleStore) List(ctx context.Context, limit, offset int) (Page[detection.Rule], error) {
	limit, offset = ClampPageParams(limit, offset)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM rf_detection_rules`).Scan(&total); err != nil {
		return Page[detection.Rule]{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, enabled, metric, operator, threshold, severity,
		       cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
		FROM rf_detection_rules
		ORDER BY enabled DESC, updated_at DESC, name ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
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

func (s *DetectionRuleStore) ListEnabled(ctx context.Context) ([]detection.Rule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, enabled, metric, operator, threshold, severity,
		       cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
		FROM rf_detection_rules
		WHERE enabled = true
		ORDER BY threshold DESC, name ASC
		LIMIT $1
	`, MaxEnabledDetectionRules)
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

func (s *DetectionRuleStore) EnabledCount(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM rf_detection_rules WHERE enabled = true`).Scan(&count)
	return count, err
}

func (s *DetectionRuleStore) Get(ctx context.Context, id uuid.UUID) (detection.Rule, error) {
	rule, err := scanDetectionRule(s.pool.QueryRow(ctx, `
		SELECT id, name, description, enabled, metric, operator, threshold, severity,
		       cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
		FROM rf_detection_rules
		WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return detection.Rule{}, ErrDetectionRuleNotFound
	}
	return rule, err
}

func (s *DetectionRuleStore) Create(ctx context.Context, rule detection.Rule, actorID uuid.UUID) (detection.Rule, error) {
	return scanDetectionRule(s.pool.QueryRow(ctx, `
		INSERT INTO rf_detection_rules (
			name, description, enabled, metric, operator, threshold, severity,
			cooldown_seconds, auto_incident, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id, name, description, enabled, metric, operator, threshold, severity,
		          cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
	`, rule.Name, rule.Description, rule.Enabled, rule.Metric, rule.Operator, rule.Threshold,
		rule.Severity, rule.CooldownSeconds, rule.AutoIncident, actorID))
}

func (s *DetectionRuleStore) Update(ctx context.Context, rule detection.Rule, actorID uuid.UUID) (detection.Rule, error) {
	updated, err := scanDetectionRule(s.pool.QueryRow(ctx, `
		UPDATE rf_detection_rules
		SET name = $2,
		    description = $3,
		    enabled = $4,
		    metric = $5,
		    operator = $6,
		    threshold = $7,
		    severity = $8,
		    cooldown_seconds = $9,
		    auto_incident = $10,
		    updated_by = $11,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, name, description, enabled, metric, operator, threshold, severity,
		          cooldown_seconds, auto_incident, created_by, updated_by, created_at, updated_at
	`, rule.ID, rule.Name, rule.Description, rule.Enabled, rule.Metric, rule.Operator, rule.Threshold,
		rule.Severity, rule.CooldownSeconds, rule.AutoIncident, actorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return detection.Rule{}, ErrDetectionRuleNotFound
	}
	return updated, err
}

func (s *DetectionRuleStore) RecordMatchIfAllowed(ctx context.Context, rule detection.Rule, anomalyID uuid.UUID, agentID, metric string, zScore float64) (DetectionMatch, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DetectionMatch{}, false, err
	}
	defer tx.Rollback(ctx)

	lockKey := rule.ID.String() + ":" + agentID
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
				WHERE rule_id = $1 AND agent_id = $2 AND matched_at >= $3
			)
		`, rule.ID, agentID, since).Scan(&recent); err != nil {
			return DetectionMatch{}, false, err
		}
		if recent {
			return DetectionMatch{}, false, nil
		}
	}

	var match DetectionMatch
	err = tx.QueryRow(ctx, `
		INSERT INTO rf_detection_matches (rule_id, anomaly_id, agent_id, metric, z_score)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (rule_id, anomaly_id) DO NOTHING
		RETURNING id, rule_id, anomaly_id, incident_id, agent_id, metric, z_score, matched_at
	`, rule.ID, anomalyID, agentID, metric, zScore).Scan(
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

func (s *DetectionRuleStore) AttachIncident(ctx context.Context, matchIDs []uuid.UUID, incidentID uuid.UUID) error {
	if len(matchIDs) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE rf_detection_matches
		SET incident_id = $2
		WHERE id = ANY($1::uuid[])
	`, matchIDs, incidentID)
	return err
}

type detectionRuleScanner interface {
	Scan(dest ...interface{}) error
}

func scanDetectionRule(row detectionRuleScanner) (detection.Rule, error) {
	var rule detection.Rule
	err := row.Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.Enabled, &rule.Metric, &rule.Operator,
		&rule.Threshold, &rule.Severity, &rule.CooldownSeconds, &rule.AutoIncident,
		&rule.CreatedBy, &rule.UpdatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
}
