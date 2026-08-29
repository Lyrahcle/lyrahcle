package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAgentIDTaken = errors.New("store: agent_id already has a key")
var ErrAgentKeyNotFound = errors.New("store: agent key not found or revoked")

type AgentKey struct {
	ID      uuid.UUID
	AgentID string
	KeyHash string
	Revoked bool
}

type AgentKeyStore struct {
	pool *pgxpool.Pool
}

func NewAgentKeyStore(pool *pgxpool.Pool) *AgentKeyStore {
	return &AgentKeyStore{pool: pool}
}

type AgentKeySummary struct {
	AgentID    string           `json:"agent_id"`
	CreatedAt  time.Time        `json:"created_at"`
	LastUsedAt *time.Time       `json:"last_used_at"`
	Revoked    bool             `json:"revoked"`
	Device     *AgentDeviceInfo `json:"device,omitempty"`
}

type AgentDeviceInfo struct {
	Driver       string    `json:"driver"`
	Hardware     string    `json:"hardware"`
	Manufacturer string    `json:"manufacturer"`
	Product      string    `json:"product"`
	Serial       string    `json:"serial"`
	RX           bool      `json:"rx"`
	TX           bool      `json:"tx"`
	ReportedAt   time.Time `json:"reported_at"`
}

func (s *AgentKeyStore) List(ctx context.Context) ([]AgentKeySummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT agent_id, created_at, last_used_at, revoked_at IS NOT NULL,
			device_driver, device_hardware, device_manufacturer, device_product,
			device_serial, device_rx, device_tx, device_reported_at
		FROM agent_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentKeySummary
	for rows.Next() {
		var summary AgentKeySummary
		var driver, hardware, manufacturer, product, serial *string
		var rx, tx *bool
		var reportedAt *time.Time
		if err := rows.Scan(
			&summary.AgentID, &summary.CreatedAt, &summary.LastUsedAt, &summary.Revoked,
			&driver, &hardware, &manufacturer, &product, &serial, &rx, &tx, &reportedAt,
		); err != nil {
			return nil, err
		}
		if reportedAt != nil {
			summary.Device = &AgentDeviceInfo{
				Driver:       deref(driver),
				Hardware:     deref(hardware),
				Manufacturer: deref(manufacturer),
				Product:      deref(product),
				Serial:       deref(serial),
				RX:           rx != nil && *rx,
				TX:           tx != nil && *tx,
				ReportedAt:   *reportedAt,
			}
		}
		out = append(out, summary)
	}
	return out, rows.Err()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *AgentKeyStore) ReportDevice(ctx context.Context, agentID string, info AgentDeviceInfo) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_keys SET
			device_driver = $1, device_hardware = $2, device_manufacturer = $3,
			device_product = $4, device_serial = $5, device_rx = $6, device_tx = $7,
			device_reported_at = now()
		WHERE agent_id = $8 AND revoked_at IS NULL
	`, info.Driver, info.Hardware, info.Manufacturer, info.Product, info.Serial, info.RX, info.TX, agentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentKeyNotFound
	}
	return nil
}

func (s *AgentKeyStore) Create(ctx context.Context, agentID, keyHash string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO agent_keys (agent_id, key_hash)
		VALUES ($1, $2)
	`, agentID, keyHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAgentIDTaken
		}
		return err
	}
	return nil
}

func (s *AgentKeyStore) AgentIDForHash(ctx context.Context, keyHash string) (string, error) {
	var agentID string
	err := s.pool.QueryRow(ctx, `
		SELECT agent_id FROM agent_keys
		WHERE key_hash = $1 AND revoked_at IS NULL
	`, keyHash).Scan(&agentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAgentKeyNotFound
		}
		return "", err
	}

	_, _ = s.pool.Exec(ctx, `UPDATE agent_keys SET last_used_at = now() WHERE key_hash = $1`, keyHash)

	return agentID, nil
}

func (s *AgentKeyStore) Revoke(ctx context.Context, agentID string) (time.Time, error) {
	var revokedAt time.Time
	err := s.pool.QueryRow(ctx, `
		UPDATE agent_keys SET revoked_at = now()
		WHERE agent_id = $1 AND revoked_at IS NULL
		RETURNING revoked_at
	`, agentID).Scan(&revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrAgentKeyNotFound
		}
		return time.Time{}, err
	}
	return revokedAt, nil
}
