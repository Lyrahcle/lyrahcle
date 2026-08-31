package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrFleetRolloutNotFound = errors.New("store: fleet rollout not found")
var ErrFleetRolloutState = errors.New("store: invalid fleet rollout state")

type FleetRolloutStore struct{ pool *pgxpool.Pool }

type FleetRollout struct {
	ID                      uuid.UUID            `json:"id"`
	OrganizationID          uuid.UUID            `json:"organization_id"`
	Name                    string               `json:"name"`
	Settings                AgentConfigSettings  `json:"config"`
	BatchSize               int                  `json:"batch_size"`
	FailureThresholdPercent int                  `json:"failure_threshold_percent"`
	SoakSeconds             int                  `json:"soak_seconds"`
	AckTimeoutSeconds       int                  `json:"ack_timeout_seconds"`
	Status                  string               `json:"status"`
	CreatedBy               *uuid.UUID           `json:"created_by,omitempty"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
	Targets                 []FleetRolloutTarget `json:"targets,omitempty"`
}

type FleetRolloutTarget struct {
	AgentID         string     `json:"agent_id"`
	Ordinal         int        `json:"ordinal"`
	Status          string     `json:"status"`
	ConfigVersion   *int64     `json:"config_version,omitempty"`
	PreviousVersion *int64     `json:"previous_version,omitempty"`
	IssuedAt        *time.Time `json:"issued_at,omitempty"`
}

type FleetTargetEligibility struct {
	AgentID         string `json:"agent_id"`
	Eligible        bool   `json:"eligible"`
	Reason          string `json:"reason,omitempty"`
	Compatibility   string `json:"compatibility,omitempty"`
	FrequencyMinHz  int64  `json:"frequency_min_hz,omitempty"`
	FrequencyMaxHz  int64  `json:"frequency_max_hz,omitempty"`
	IntegrityStatus string `json:"integrity_status,omitempty"`
}

func NewFleetRolloutStore(pool *pgxpool.Pool) *FleetRolloutStore {
	return &FleetRolloutStore{pool: pool}
}

func (s *FleetRolloutStore) Create(ctx context.Context, orgID uuid.UUID, name string, settings AgentConfigSettings, batchSize, failureThreshold, soakSeconds, ackTimeoutSeconds int, agentIDs []string, createdBy *uuid.UUID) (FleetRollout, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FleetRollout{}, err
	}
	defer tx.Rollback(ctx)
	var r FleetRollout
	r.OrganizationID, r.Name, r.Settings, r.BatchSize, r.FailureThresholdPercent, r.SoakSeconds, r.AckTimeoutSeconds, r.Status, r.CreatedBy = orgID, name, settings, batchSize, failureThreshold, soakSeconds, ackTimeoutSeconds, "draft", createdBy
	err = tx.QueryRow(ctx, `INSERT INTO agent_fleet_rollouts (organization_id,name,frequency_hz,capture_interval_seconds,satellite_id,batch_size,failure_threshold_percent,soak_seconds,ack_timeout_seconds,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,created_at,updated_at`, orgID, name, settings.FrequencyHz, settings.CaptureIntervalSeconds, settings.SatelliteID, batchSize, failureThreshold, soakSeconds, ackTimeoutSeconds, createdBy).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return FleetRollout{}, err
	}
	if len(agentIDs) > 0 {
		tag, err := tx.Exec(ctx, `
			WITH requested(agent_id, ordinal) AS (
				SELECT agent_id, (ordinality - 1)::integer
				FROM unnest($3::text[]) WITH ORDINALITY AS requested(agent_id, ordinality)
			)
			INSERT INTO agent_fleet_rollout_targets (rollout_id, organization_id, agent_id, ordinal)
			SELECT $1, $2, i.agent_id, requested.ordinal
			FROM requested
			JOIN agent_identities i
			  ON i.agent_id = requested.agent_id
			 AND i.organization_id = $2
			 AND i.status = 'active'
		`, r.ID, orgID, agentIDs)
		if err != nil {
			return FleetRollout{}, err
		}
		if tag.RowsAffected() != int64(len(agentIDs)) {
			return FleetRollout{}, ErrAgentIdentityNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return FleetRollout{}, err
	}
	return s.Get(ctx, orgID, r.ID)
}

func (s *FleetRolloutStore) List(ctx context.Context, orgID uuid.UUID) ([]FleetRollout, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,organization_id,name,frequency_hz,capture_interval_seconds,satellite_id,batch_size,failure_threshold_percent,soak_seconds,ack_timeout_seconds,status,created_by,created_at,updated_at FROM agent_fleet_rollouts WHERE organization_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FleetRollout{}
	for rows.Next() {
		var r FleetRollout
		if err := rows.Scan(&r.ID, &r.OrganizationID, &r.Name, &r.Settings.FrequencyHz, &r.Settings.CaptureIntervalSeconds, &r.Settings.SatelliteID, &r.BatchSize, &r.FailureThresholdPercent, &r.SoakSeconds, &r.AckTimeoutSeconds, &r.Status, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *FleetRolloutStore) Get(ctx context.Context, orgID, id uuid.UUID) (FleetRollout, error) {
	var r FleetRollout
	err := s.pool.QueryRow(ctx, `SELECT id,organization_id,name,frequency_hz,capture_interval_seconds,satellite_id,batch_size,failure_threshold_percent,soak_seconds,ack_timeout_seconds,status,created_by,created_at,updated_at FROM agent_fleet_rollouts WHERE id=$1 AND organization_id=$2`, id, orgID).Scan(&r.ID, &r.OrganizationID, &r.Name, &r.Settings.FrequencyHz, &r.Settings.CaptureIntervalSeconds, &r.Settings.SatelliteID, &r.BatchSize, &r.FailureThresholdPercent, &r.SoakSeconds, &r.AckTimeoutSeconds, &r.Status, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return FleetRollout{}, ErrFleetRolloutNotFound
	}
	if err != nil {
		return FleetRollout{}, err
	}
	rows, err := s.pool.Query(ctx, `SELECT agent_id,ordinal,status,config_version,previous_version,issued_at FROM agent_fleet_rollout_targets WHERE rollout_id=$1 AND organization_id=$2 ORDER BY ordinal`, id, orgID)
	if err != nil {
		return FleetRollout{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var t FleetRolloutTarget
		if err := rows.Scan(&t.AgentID, &t.Ordinal, &t.Status, &t.ConfigVersion, &t.PreviousVersion, &t.IssuedAt); err != nil {
			return FleetRollout{}, err
		}
		r.Targets = append(r.Targets, t)
	}
	return r, rows.Err()
}

func (s *FleetRolloutStore) EvaluateTargets(ctx context.Context, orgID uuid.UUID, agentIDs []string, frequencyHz int64) ([]FleetTargetEligibility, error) {
	type identity struct {
		status       string
		driver       *string
		hardware     *string
		manufacturer *string
		product      *string
		serial       *string
		rx           *bool
		tx           *bool
		reportedAt   *time.Time
		capsJSON     []byte
	}
	rows, err := s.pool.Query(ctx, `
		SELECT agent_id,status,device_driver,device_hardware,device_manufacturer,device_product,
		       device_serial,device_rx,device_tx,device_reported_at,device_capabilities
		FROM agent_identities
		WHERE organization_id=$1 AND agent_id=ANY($2)
	`, orgID, agentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := make(map[string]identity, len(agentIDs))
	for rows.Next() {
		var agentID string
		var item identity
		if err := rows.Scan(&agentID, &item.status, &item.driver, &item.hardware, &item.manufacturer, &item.product, &item.serial, &item.rx, &item.tx, &item.reportedAt, &item.capsJSON); err != nil {
			return nil, err
		}
		found[agentID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]FleetTargetEligibility, 0, len(agentIDs))
	now := time.Now().UTC()
	for _, agentID := range agentIDs {
		result := FleetTargetEligibility{AgentID: agentID}
		item, ok := found[agentID]
		if !ok {
			result.Reason = "agent does not belong to the organization"
			out = append(out, result)
			continue
		}
		if item.status != "active" {
			result.Reason = "agent is not active"
			out = append(out, result)
			continue
		}
		if item.reportedAt == nil {
			result.Reason = "hardware capability report is missing"
			out = append(out, result)
			continue
		}
		info := AgentDeviceInfo{
			Driver: deref(item.driver), Hardware: deref(item.hardware), Manufacturer: deref(item.manufacturer),
			Product: deref(item.product), Serial: deref(item.serial), RX: item.rx != nil && *item.rx,
			TX: item.tx != nil && *item.tx, ReportedAt: *item.reportedAt,
		}
		_ = json.Unmarshal(item.capsJSON, &info.Capabilities)
		finalizeAgentDeviceInfo(&info, now)
		result.Compatibility = info.Compatibility
		result.FrequencyMinHz = info.Capabilities.FrequencyMinHz
		result.FrequencyMaxHz = info.Capabilities.FrequencyMaxHz
		result.IntegrityStatus = info.Capabilities.IntegrityStatus
		if info.Compatibility != "compatible" {
			result.Reason = info.CompatibilityReason
			out = append(out, result)
			continue
		}
		if info.Capabilities.IntegrityStatus == "changed" {
			result.Reason = "edge integrity manifest changed from its baseline"
			out = append(out, result)
			continue
		}
		if frequencyHz < info.Capabilities.FrequencyMinHz || frequencyHz > info.Capabilities.FrequencyMaxHz {
			result.Reason = fmt.Sprintf("frequency %d Hz is outside reported hardware range %d-%d Hz", frequencyHz, info.Capabilities.FrequencyMinHz, info.Capabilities.FrequencyMaxHz)
			out = append(out, result)
			continue
		}
		result.Eligible = true
		out = append(out, result)
	}
	return out, nil
}

func (s *FleetRolloutStore) SetStatus(ctx context.Context, orgID, id uuid.UUID, allowed []string, status string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE agent_fleet_rollouts SET status=$4,updated_at=now() WHERE id=$1 AND organization_id=$2 AND status=ANY($3)`, id, orgID, allowed, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrFleetRolloutState
	}
	return nil
}

func (s *FleetRolloutStore) NextPending(ctx context.Context, orgID, id uuid.UUID, limit int) ([]FleetRolloutTarget, error) {
	rows, err := s.pool.Query(ctx, `SELECT agent_id,ordinal,status,config_version,previous_version,issued_at FROM agent_fleet_rollout_targets WHERE rollout_id=$1 AND organization_id=$2 AND status='pending' ORDER BY ordinal LIMIT $3`, id, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FleetRolloutTarget{}
	for rows.Next() {
		var t FleetRolloutTarget
		if err := rows.Scan(&t.AgentID, &t.Ordinal, &t.Status, &t.ConfigVersion, &t.PreviousVersion, &t.IssuedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *FleetRolloutStore) MarkIssued(ctx context.Context, orgID, id uuid.UUID, agentID string, previousVersion *int64, configVersion int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE agent_fleet_rollout_targets SET status='issued',previous_version=$4,config_version=$5,issued_at=now(),updated_at=now() WHERE rollout_id=$1 AND organization_id=$2 AND agent_id=$3`, id, orgID, agentID, previousVersion, configVersion)
	return err
}

func (s *FleetRolloutStore) RefreshAcknowledgements(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE agent_fleet_rollout_targets t SET status=a.status,updated_at=now() FROM agent_config_acks a WHERE t.rollout_id=$1 AND t.organization_id=$2 AND a.organization_id=$2 AND t.config_version=a.version AND t.agent_id=a.agent_id AND t.status='issued' AND a.status IN ('applied','rejected')`, id, orgID)
	return err
}

func (s *FleetRolloutStore) Counts(ctx context.Context, orgID, id uuid.UUID) (pending, issued, applied, rejected int, err error) {
	err = s.pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE status='pending'),count(*) FILTER(WHERE status='issued'),count(*) FILTER(WHERE status='applied'),count(*) FILTER(WHERE status='rejected') FROM agent_fleet_rollout_targets WHERE rollout_id=$1 AND organization_id=$2`, id, orgID).Scan(&pending, &issued, &applied, &rejected)
	return
}

func (s *FleetRolloutStore) OverdueIssuedCount(ctx context.Context, orgID, id uuid.UUID, ackTimeoutSeconds int) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM agent_fleet_rollout_targets
		WHERE rollout_id=$1 AND organization_id=$2 AND status='issued'
		  AND issued_at IS NOT NULL
		  AND issued_at <= now() - ($3 * interval '1 second')
	`, id, orgID, ackTimeoutSeconds).Scan(&count)
	return count, err
}

func (s *FleetRolloutStore) MarkRolledBack(ctx context.Context, orgID, id uuid.UUID, agentID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE agent_fleet_rollout_targets SET status='rolled_back',updated_at=now() WHERE rollout_id=$1 AND organization_id=$2 AND agent_id=$3`, id, orgID, agentID)
	return err
}

func (s *FleetRolloutStore) LatestIssuedAt(ctx context.Context, orgID, id uuid.UUID) (*time.Time, error) {
	var issuedAt *time.Time
	err := s.pool.QueryRow(ctx, `SELECT max(issued_at) FROM agent_fleet_rollout_targets WHERE rollout_id=$1 AND organization_id=$2`, id, orgID).Scan(&issuedAt)
	return issuedAt, err
}
