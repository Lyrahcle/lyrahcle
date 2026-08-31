package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

type AgentDeviceCapabilities struct {
	Architecture    string   `json:"architecture,omitempty"`
	OS              string   `json:"os,omitempty"`
	DriverVersion   string   `json:"driver_version,omitempty"`
	FirmwareVersion string   `json:"firmware_version,omitempty"`
	FrequencyMinHz  int64    `json:"frequency_min_hz,omitempty"`
	FrequencyMaxHz  int64    `json:"frequency_max_hz,omitempty"`
	SampleRatesHz   []int64  `json:"sample_rates_hz,omitempty"`
	BandwidthsHz    []int64  `json:"bandwidths_hz,omitempty"`
	GainElements    []string `json:"gain_elements,omitempty"`
	Channels        int      `json:"channels,omitempty"`
	GPS             bool     `json:"gps"`
	PPS             bool     `json:"pps"`
	TemperatureC    *float64 `json:"temperature_c,omitempty"`
	SampleDrops     int64    `json:"sample_drops"`
}

type AgentDeviceInfo struct {
	Driver              string                  `json:"driver"`
	Hardware            string                  `json:"hardware"`
	Manufacturer        string                  `json:"manufacturer"`
	Product             string                  `json:"product"`
	Serial              string                  `json:"serial"`
	RX                  bool                    `json:"rx"`
	TX                  bool                    `json:"tx"`
	Capabilities        AgentDeviceCapabilities `json:"capabilities"`
	ReportedAt          time.Time               `json:"reported_at"`
	Compatibility       string                  `json:"compatibility"`
	CompatibilityReason string                  `json:"compatibility_reason"`
}

func finalizeAgentDeviceInfo(info *AgentDeviceInfo, now time.Time) {
	if info == nil {
		return
	}
	if !info.ReportedAt.IsZero() && now.Sub(info.ReportedAt) > 15*time.Minute {
		info.Compatibility = "stale"
		info.CompatibilityReason = "hardware capability report is older than 15 minutes"
		return
	}
	driver := strings.ToLower(info.Driver + " " + info.Hardware)
	known := false
	for _, marker := range []string{"rtlsdr", "rtl-sdr", "rtl283", "hackrf", "bladerf", "uhd", "usrp"} {
		if strings.Contains(driver, marker) {
			known = true
			break
		}
	}
	if !info.RX || !known {
		info.Compatibility = "unsupported"
		if !info.RX {
			info.CompatibilityReason = "device does not report RX capability"
		} else {
			info.CompatibilityReason = "driver is not in the v1 software compatibility baseline"
		}
		return
	}
	c := info.Capabilities
	if c.FrequencyMinHz <= 0 || c.FrequencyMaxHz <= c.FrequencyMinHz || len(c.SampleRatesHz) == 0 || c.Channels <= 0 {
		info.Compatibility = "degraded"
		info.CompatibilityReason = "device is recognized but the capability report is incomplete"
		return
	}
	info.Compatibility = "compatible"
	info.CompatibilityReason = "software capability report satisfies the v1 baseline; physical lab validation is tracked separately"
}

func (s *AgentKeyStore) List(ctx context.Context) ([]AgentKeySummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT agent_id, created_at, last_used_at, revoked_at IS NOT NULL,
			device_driver, device_hardware, device_manufacturer, device_product,
			device_serial, device_rx, device_tx, device_reported_at, device_capabilities
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
		var capabilitiesJSON []byte
		if err := rows.Scan(
			&summary.AgentID, &summary.CreatedAt, &summary.LastUsedAt, &summary.Revoked,
			&driver, &hardware, &manufacturer, &product, &serial, &rx, &tx, &reportedAt, &capabilitiesJSON,
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
			_ = json.Unmarshal(capabilitiesJSON, &summary.Device.Capabilities)
			finalizeAgentDeviceInfo(summary.Device, time.Now().UTC())
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
	capabilities, err := json.Marshal(info.Capabilities)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_keys SET
			device_driver = $1, device_hardware = $2, device_manufacturer = $3,
			device_product = $4, device_serial = $5, device_rx = $6, device_tx = $7,
			device_reported_at = now(), device_capabilities = $8::jsonb
		WHERE agent_id = $9 AND revoked_at IS NULL
	`, info.Driver, info.Hardware, info.Manufacturer, info.Product, info.Serial, info.RX, info.TX, string(capabilities), agentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentKeyNotFound
	}
	return nil
}

func (s *AgentKeyStore) ReportDeviceInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string, info AgentDeviceInfo) error {
	capabilities, err := json.Marshal(info.Capabilities)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_keys SET
			device_driver = $1, device_hardware = $2, device_manufacturer = $3,
			device_product = $4, device_serial = $5, device_rx = $6, device_tx = $7,
			device_reported_at = now(), device_capabilities = $8::jsonb
		WHERE agent_id = $9 AND organization_id = $10 AND revoked_at IS NULL
	`, info.Driver, info.Hardware, info.Manufacturer, info.Product, info.Serial, info.RX, info.TX, string(capabilities), agentID, organizationID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentKeyNotFound
	}
	return nil
}

func (s *AgentKeyStore) Create(ctx context.Context, agentID, keyHash string) error {
	return s.CreateInOrganization(ctx, DefaultOrganizationID, agentID, keyHash)
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

	if _, err := s.pool.Exec(ctx, `UPDATE agent_keys SET last_used_at = now() WHERE key_hash = $1`, keyHash); err != nil {
		return "", err
	}

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

func (s *AgentKeyStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]AgentKeySummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT agent_id, created_at, last_used_at, revoked_at IS NOT NULL,
		       device_driver, device_hardware, device_manufacturer, device_product,
		       device_serial, device_rx, device_tx, device_reported_at, device_capabilities
		FROM agent_keys
		WHERE organization_id=$1
		ORDER BY created_at DESC
	`, organizationID)
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
		var capabilitiesJSON []byte
		if err := rows.Scan(&summary.AgentID, &summary.CreatedAt, &summary.LastUsedAt, &summary.Revoked,
			&driver, &hardware, &manufacturer, &product, &serial, &rx, &tx, &reportedAt, &capabilitiesJSON); err != nil {
			return nil, err
		}
		if reportedAt != nil {
			summary.Device = &AgentDeviceInfo{Driver: deref(driver), Hardware: deref(hardware), Manufacturer: deref(manufacturer),
				Product: deref(product), Serial: deref(serial), RX: rx != nil && *rx, TX: tx != nil && *tx, ReportedAt: *reportedAt}
			_ = json.Unmarshal(capabilitiesJSON, &summary.Device.Capabilities)
			finalizeAgentDeviceInfo(summary.Device, time.Now().UTC())
		}
		out = append(out, summary)
	}
	return out, rows.Err()
}

func (s *AgentKeyStore) CreateInOrganization(ctx context.Context, organizationID uuid.UUID, agentID, keyHash string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO agent_keys (organization_id, agent_id, key_hash) VALUES ($1,$2,$3)`, organizationID, agentID, keyHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAgentIDTaken
		}
	}
	return err
}

func (s *AgentKeyStore) RevokeInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string) (time.Time, error) {
	var revokedAt time.Time
	err := s.pool.QueryRow(ctx, `
		UPDATE agent_keys SET revoked_at=now()
		WHERE agent_id=$1 AND organization_id=$2 AND revoked_at IS NULL
		RETURNING revoked_at
	`, agentID, organizationID).Scan(&revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrAgentKeyNotFound
	}
	return revokedAt, err
}

func (s *AgentKeyStore) OrganizationForAgent(ctx context.Context, agentID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM agent_keys WHERE agent_id=$1 AND revoked_at IS NULL`, agentID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrAgentKeyNotFound
	}
	return id, err
}
