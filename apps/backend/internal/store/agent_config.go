package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAgentConfigNotFound = errors.New("store: agent config not found")
var ErrAgentConfigInactive = errors.New("store: agent is not active")
var ErrAgentConfigAckConflict = errors.New("store: agent config acknowledgement conflict")

type AgentConfigStore struct {
	pool *pgxpool.Pool
}

type AgentConfigSettings struct {
	FrequencyHz            int64      `json:"frequency_hz"`
	CaptureIntervalSeconds int        `json:"capture_interval_seconds"`
	SatelliteID            *uuid.UUID `json:"satellite_id,omitempty"`
}

type AgentConfigVersion struct {
	ID         uuid.UUID           `json:"id"`
	AgentID    string              `json:"agent_id"`
	Version    int64               `json:"version"`
	Settings   AgentConfigSettings `json:"config"`
	Payload    []byte              `json:"-"`
	Signature  []byte              `json:"-"`
	Checksum   string              `json:"checksum_sha256"`
	SigningKID string              `json:"signing_kid,omitempty"`
	CreatedBy  *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
	AckStatus  string              `json:"ack_status,omitempty"`
	AckMessage string              `json:"ack_message,omitempty"`
	AckedAt    *time.Time          `json:"acked_at,omitempty"`
}

type AgentConfigSignResult struct {
	Payload    []byte
	Signature  []byte
	Checksum   string
	SigningKID string
}

type AgentConfigSignFunc func(version int64, settings AgentConfigSettings, issuedAt time.Time) (AgentConfigSignResult, error)

func NewAgentConfigStore(pool *pgxpool.Pool) *AgentConfigStore {
	return &AgentConfigStore{pool: pool}
}

func (s *AgentConfigStore) Create(ctx context.Context, agentID string, settings AgentConfigSettings, createdBy *uuid.UUID, sign AgentConfigSignFunc) (AgentConfigVersion, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AgentConfigVersion{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "agent-config:"+agentID); err != nil {
		return AgentConfigVersion{}, err
	}
	var organizationID uuid.UUID
	var status string
	if err := tx.QueryRow(ctx, `SELECT organization_id, status FROM agent_identities WHERE agent_id = $1 FOR UPDATE`, agentID).Scan(&organizationID, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentConfigVersion{}, ErrAgentIdentityNotFound
		}
		return AgentConfigVersion{}, err
	}
	if status != "active" {
		return AgentConfigVersion{}, ErrAgentConfigInactive
	}
	var version int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM agent_config_versions WHERE agent_id = $1 AND organization_id = $2`, agentID, organizationID).Scan(&version); err != nil {
		return AgentConfigVersion{}, err
	}
	issuedAt := time.Now().UTC()
	signed, err := sign(version, settings, issuedAt)
	if err != nil {
		return AgentConfigVersion{}, err
	}
	var item AgentConfigVersion
	item.AgentID = agentID
	item.Settings = settings
	item.Version = version
	item.Payload = signed.Payload
	item.Signature = signed.Signature
	item.Checksum = signed.Checksum
	item.SigningKID = signed.SigningKID
	item.CreatedBy = createdBy
	if err := tx.QueryRow(ctx, `
		INSERT INTO agent_config_versions (
			organization_id, agent_id, version, frequency_hz, capture_interval_seconds, satellite_id, payload, signature, checksum_sha256, signing_kid, created_by, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at
	`, organizationID, agentID, version, settings.FrequencyHz, settings.CaptureIntervalSeconds, settings.SatelliteID, signed.Payload, signed.Signature, signed.Checksum, signed.SigningKID, createdBy, issuedAt).Scan(&item.ID, &item.CreatedAt); err != nil {
		return AgentConfigVersion{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO agent_config_state (organization_id, agent_id, desired_version, acknowledged_version, ack_status, ack_message, acked_at, updated_at)
		VALUES ($1, $2, $3, NULL, 'pending', '', NULL, now())
		ON CONFLICT (agent_id) DO UPDATE SET
			organization_id = EXCLUDED.organization_id,
			desired_version = EXCLUDED.desired_version,
			acknowledged_version = NULL,
			ack_status = 'pending',
			ack_message = '',
			acked_at = NULL,
			updated_at = now()
	`, organizationID, agentID, version); err != nil {
		return AgentConfigVersion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AgentConfigVersion{}, err
	}
	item.AckStatus = "pending"
	return item, nil
}

func (s *AgentConfigStore) Desired(ctx context.Context, agentID string) (AgentConfigVersion, error) {
	var item AgentConfigVersion
	item.AgentID = agentID
	err := s.pool.QueryRow(ctx, `
		SELECT v.id, v.version, v.frequency_hz, v.capture_interval_seconds, v.satellite_id, v.payload, v.signature,
		       v.checksum_sha256, v.signing_kid, v.created_by, v.created_at, s.ack_status, s.ack_message, s.acked_at
		FROM agent_config_state s
		JOIN agent_config_versions v ON v.agent_id = s.agent_id AND v.version = s.desired_version
		WHERE s.agent_id = $1
	`, agentID).Scan(
		&item.ID, &item.Version, &item.Settings.FrequencyHz, &item.Settings.CaptureIntervalSeconds, &item.Settings.SatelliteID,
		&item.Payload, &item.Signature, &item.Checksum, &item.SigningKID, &item.CreatedBy, &item.CreatedAt,
		&item.AckStatus, &item.AckMessage, &item.AckedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentConfigVersion{}, ErrAgentConfigNotFound
		}
		return AgentConfigVersion{}, err
	}
	return item, nil
}

func (s *AgentConfigStore) History(ctx context.Context, agentID string, limit int) ([]AgentConfigVersion, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT v.id, v.version, v.frequency_hz, v.capture_interval_seconds, v.satellite_id, v.checksum_sha256, v.signing_kid, v.created_by, v.created_at,
		       COALESCE(a.status, ''), COALESCE(a.message, ''), a.created_at
		FROM agent_config_versions v
		LEFT JOIN agent_config_acks a ON a.agent_id = v.agent_id AND a.version = v.version
		WHERE v.agent_id = $1
		ORDER BY v.version DESC
		LIMIT $2
	`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AgentConfigVersion, 0)
	for rows.Next() {
		var item AgentConfigVersion
		item.AgentID = agentID
		if err := rows.Scan(&item.ID, &item.Version, &item.Settings.FrequencyHz, &item.Settings.CaptureIntervalSeconds, &item.Settings.SatelliteID, &item.Checksum, &item.SigningKID, &item.CreatedBy, &item.CreatedAt, &item.AckStatus, &item.AckMessage, &item.AckedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *AgentConfigStore) SettingsForVersion(ctx context.Context, agentID string, version int64) (AgentConfigSettings, error) {
	var settings AgentConfigSettings
	err := s.pool.QueryRow(ctx, `
		SELECT frequency_hz, capture_interval_seconds, satellite_id
		FROM agent_config_versions
		WHERE agent_id = $1 AND version = $2
	`, agentID, version).Scan(&settings.FrequencyHz, &settings.CaptureIntervalSeconds, &settings.SatelliteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentConfigSettings{}, ErrAgentConfigNotFound
		}
		return AgentConfigSettings{}, err
	}
	return settings, nil
}

func (s *AgentConfigStore) Ack(ctx context.Context, agentID string, version int64, status, message, checksum string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var organizationID uuid.UUID
	var expectedChecksum string
	if err := tx.QueryRow(ctx, `
		SELECT organization_id, checksum_sha256 FROM agent_config_versions
		WHERE agent_id = $1 AND version = $2
	`, agentID, version).Scan(&organizationID, &expectedChecksum); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAgentConfigNotFound
		}
		return err
	}
	if checksum != expectedChecksum {
		return ErrAgentConfigAckConflict
	}
	var existingStatus, existingMessage, existingChecksum string
	err = tx.QueryRow(ctx, `SELECT status, message, checksum_sha256 FROM agent_config_acks WHERE agent_id = $1 AND version = $2 FOR UPDATE`, agentID, version).Scan(&existingStatus, &existingMessage, &existingChecksum)
	if err == nil {
		if existingStatus != status || existingMessage != message || existingChecksum != checksum {
			return ErrAgentConfigAckConflict
		}
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO agent_config_acks (organization_id, agent_id, version, status, message, checksum_sha256)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, organizationID, agentID, version, status, message, checksum); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE agent_config_state
		SET acknowledged_version = $2, ack_status = $3, ack_message = $4, acked_at = now(), updated_at = now()
		WHERE agent_id = $1 AND desired_version = $2 AND organization_id = $5
	`, agentID, version, status, message, organizationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AgentConfigStore) DesiredInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string) (AgentConfigVersion, error) {
	var ok bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM agent_identities WHERE agent_id=$1 AND organization_id=$2)`, agentID, organizationID).Scan(&ok); err != nil {
		return AgentConfigVersion{}, err
	}
	if !ok {
		return AgentConfigVersion{}, ErrAgentIdentityNotFound
	}
	return s.Desired(ctx, agentID)
}

func (s *AgentConfigStore) HistoryInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string, limit int) ([]AgentConfigVersion, error) {
	var ok bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM agent_identities WHERE agent_id=$1 AND organization_id=$2)`, agentID, organizationID).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAgentIdentityNotFound
	}
	return s.History(ctx, agentID, limit)
}

func (s *AgentConfigStore) SettingsForVersionInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string, version int64) (AgentConfigSettings, error) {
	var ok bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM agent_identities WHERE agent_id=$1 AND organization_id=$2)`, agentID, organizationID).Scan(&ok); err != nil {
		return AgentConfigSettings{}, err
	}
	if !ok {
		return AgentConfigSettings{}, ErrAgentIdentityNotFound
	}
	return s.SettingsForVersion(ctx, agentID, version)
}

func (s *AgentConfigStore) CreateInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string, settings AgentConfigSettings, createdBy *uuid.UUID, sign AgentConfigSignFunc) (AgentConfigVersion, error) {
	var ok bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM agent_identities WHERE agent_id=$1 AND organization_id=$2)`, agentID, organizationID).Scan(&ok); err != nil {
		return AgentConfigVersion{}, err
	}
	if !ok {
		return AgentConfigVersion{}, ErrAgentIdentityNotFound
	}
	return s.Create(ctx, agentID, settings, createdBy, sign)
}

func (s *AgentConfigStore) ClearDesiredInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM agent_config_state WHERE organization_id=$1 AND agent_id=$2`, organizationID, agentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentConfigNotFound
	}
	return nil
}
