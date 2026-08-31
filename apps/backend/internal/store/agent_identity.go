package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAgentIdentityNotFound = errors.New("store: agent identity not found")
var ErrAgentEnrollmentInvalid = errors.New("store: enrollment token invalid or expired")
var ErrAgentIdentityInactive = errors.New("store: agent identity is not active")
var ErrAgentAlreadyEnrolled = errors.New("store: agent identity already has an active certificate")
var ErrAgentCertificateNotFound = errors.New("store: agent certificate not found")
var ErrAgentStatusTransition = errors.New("store: invalid agent status transition")
var ErrAgentStationNotConfigured = errors.New("store: agent station is not configured")

type AgentIdentityStore struct {
	pool *pgxpool.Pool
}

type AgentIdentity struct {
	AgentID         string            `json:"agent_id"`
	OrganizationID  uuid.UUID         `json:"organization_id"`
	Status          string            `json:"status"`
	FirstSeenAt     *time.Time        `json:"first_seen_at"`
	LastSeenAt      *time.Time        `json:"last_seen_at"`
	AgentVersion    string            `json:"agent_version"`
	FirmwareVersion string            `json:"firmware_version"`
	Certificate     *AgentCertificate `json:"certificate,omitempty"`
	Device          *AgentDeviceInfo  `json:"device,omitempty"`
	Station         *AgentStation     `json:"station,omitempty"`
}

type AgentStation struct {
	LatitudeDeg  float64 `json:"latitude_deg"`
	LongitudeDeg float64 `json:"longitude_deg"`
	AltitudeM    float64 `json:"altitude_m"`
}

type AgentCertificate struct {
	ID                uuid.UUID  `json:"id"`
	AgentID           string     `json:"agent_id"`
	SerialNumber      string     `json:"serial_number"`
	FingerprintSHA256 string     `json:"fingerprint_sha256"`
	CertificatePEM    string     `json:"-"`
	NotBefore         time.Time  `json:"not_before"`
	NotAfter          time.Time  `json:"not_after"`
	Status            string     `json:"status"`
	LastSeenAt        *time.Time `json:"last_seen_at"`
}

func NewAgentIdentityStore(pool *pgxpool.Pool) *AgentIdentityStore {
	return &AgentIdentityStore{pool: pool}
}

func (s *AgentIdentityStore) CreateEnrollmentToken(ctx context.Context, agentID, tokenHash string, expiresAt time.Time) error {
	return s.CreateEnrollmentTokenInOrganization(ctx, DefaultOrganizationID, agentID, tokenHash, expiresAt)
}

func (s *AgentIdentityStore) AgentIDForEnrollmentToken(ctx context.Context, tokenHash string) (string, error) {
	var agentID string
	err := s.pool.QueryRow(ctx, `
		SELECT t.agent_id
		FROM agent_enrollment_tokens t
		JOIN agent_identities i ON i.agent_id = t.agent_id
		WHERE t.token_hash = $1 AND t.used_at IS NULL AND t.expires_at > now()
		  AND i.status NOT IN ('revoked', 'quarantined')
	`, tokenHash).Scan(&agentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAgentEnrollmentInvalid
		}
		return "", err
	}
	return agentID, nil
}

func (s *AgentIdentityStore) ConsumeEnrollmentToken(ctx context.Context, tokenHash string, cert AgentCertificate, agentVersion, firmwareVersion string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var agentID string
	var organizationID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `
		SELECT t.agent_id, t.organization_id, i.status
		FROM agent_enrollment_tokens t
		JOIN agent_identities i ON i.agent_id = t.agent_id
		WHERE t.token_hash = $1 AND t.used_at IS NULL AND t.expires_at > now()
		FOR UPDATE OF t, i
	`, tokenHash).Scan(&agentID, &organizationID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAgentEnrollmentInvalid
		}
		return "", err
	}
	if status == "revoked" || status == "quarantined" {
		return "", ErrAgentIdentityInactive
	}

	cert.AgentID = agentID
	if err := insertAgentCertificate(ctx, tx, organizationID, cert); err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		UPDATE agent_enrollment_tokens SET used_at = now() WHERE token_hash = $1
	`, tokenHash)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		UPDATE agent_identities SET
			status = 'active',
			first_seen_at = COALESCE(first_seen_at, now()),
			last_seen_at = now(),
			agent_version = $2,
			firmware_version = $3,
			updated_at = now()
		WHERE agent_id = $1
	`, agentID, agentVersion, firmwareVersion)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return agentID, nil
}

func (s *AgentIdentityStore) RotateCertificate(ctx context.Context, agentID, currentFingerprint string, cert AgentCertificate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var organizationID uuid.UUID
	var status string
	err = tx.QueryRow(ctx, `SELECT organization_id, status FROM agent_identities WHERE agent_id = $1 FOR UPDATE`, agentID).Scan(&organizationID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAgentIdentityNotFound
		}
		return err
	}
	if status != "active" {
		return ErrAgentIdentityInactive
	}

	var oldID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM agent_certificates
		WHERE agent_id = $1 AND fingerprint_sha256 = $2 AND status = 'active' AND not_after > now()
		FOR UPDATE
	`, agentID, currentFingerprint).Scan(&oldID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAgentCertificateNotFound
		}
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE agent_certificates
		SET status = 'superseded', superseded_at = now()
		WHERE id = $1
	`, oldID)
	if err != nil {
		return err
	}

	cert.AgentID = agentID
	if err := insertAgentCertificate(ctx, tx, organizationID, cert); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE agent_identities SET updated_at = now() WHERE agent_id = $1`, agentID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertAgentCertificate(ctx context.Context, tx pgx.Tx, organizationID uuid.UUID, cert AgentCertificate) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO agent_certificates (
			organization_id, agent_id, serial_number, fingerprint_sha256, certificate_pem, not_before, not_after, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
	`, organizationID, cert.AgentID, cert.SerialNumber, cert.FingerprintSHA256, cert.CertificatePEM, cert.NotBefore, cert.NotAfter)
	return err
}

func (s *AgentIdentityStore) AgentIDForCertificate(ctx context.Context, fingerprint string) (string, error) {
	var agentID string
	err := s.pool.QueryRow(ctx, `
		SELECT c.agent_id
		FROM agent_certificates c
		JOIN agent_identities i ON i.agent_id = c.agent_id
		WHERE c.fingerprint_sha256 = $1
		  AND c.status = 'active'
		  AND c.not_before <= now()
		  AND c.not_after > now()
		  AND i.status = 'active'
	`, fingerprint).Scan(&agentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAgentCertificateNotFound
		}
		return "", err
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE agent_certificates SET last_seen_at = now()
		WHERE fingerprint_sha256 = $1 AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
	`, fingerprint); err != nil {
		return "", err
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE agent_identities SET last_seen_at = now(), updated_at = now()
		WHERE agent_id = $1 AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
	`, agentID); err != nil {
		return "", err
	}
	return agentID, nil
}

func (s *AgentIdentityStore) List(ctx context.Context) ([]AgentIdentity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			i.agent_id, i.status, i.first_seen_at, i.last_seen_at, i.agent_version, i.firmware_version,
			i.device_driver, i.device_hardware, i.device_manufacturer, i.device_product,
			i.device_serial, i.device_rx, i.device_tx, i.device_reported_at,
			i.station_latitude_deg, i.station_longitude_deg, i.station_altitude_m,
			c.id, c.serial_number, c.fingerprint_sha256, c.not_before, c.not_after, c.status, c.last_seen_at
		FROM agent_identities i
		LEFT JOIN agent_certificates c ON c.agent_id = i.agent_id AND c.status = 'active'
		ORDER BY i.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AgentIdentity, 0)
	for rows.Next() {
		var item AgentIdentity
		var driver, hardware, manufacturer, product, serial *string
		var rx, tx *bool
		var reportedAt *time.Time
		var stationLat, stationLon, stationAlt *float64
		var certID *uuid.UUID
		var certSerial, fingerprint, certStatus *string
		var notBefore, notAfter, certLastSeen *time.Time
		if err := rows.Scan(
			&item.AgentID, &item.Status, &item.FirstSeenAt, &item.LastSeenAt, &item.AgentVersion, &item.FirmwareVersion,
			&driver, &hardware, &manufacturer, &product, &serial, &rx, &tx, &reportedAt,
			&stationLat, &stationLon, &stationAlt,
			&certID, &certSerial, &fingerprint, &notBefore, &notAfter, &certStatus, &certLastSeen,
		); err != nil {
			return nil, err
		}
		if stationLat != nil && stationLon != nil && stationAlt != nil {
			item.Station = &AgentStation{LatitudeDeg: *stationLat, LongitudeDeg: *stationLon, AltitudeM: *stationAlt}
		}
		if reportedAt != nil {
			item.Device = &AgentDeviceInfo{
				Driver: deref(driver), Hardware: deref(hardware), Manufacturer: deref(manufacturer), Product: deref(product),
				Serial: deref(serial), RX: rx != nil && *rx, TX: tx != nil && *tx, ReportedAt: *reportedAt,
			}
		}
		if certID != nil {
			item.Certificate = &AgentCertificate{
				ID: *certID, AgentID: item.AgentID, SerialNumber: deref(certSerial), FingerprintSHA256: deref(fingerprint),
				NotBefore: *notBefore, NotAfter: *notAfter, Status: deref(certStatus), LastSeenAt: certLastSeen,
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *AgentIdentityStore) SetStatus(ctx context.Context, agentID, status string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM agent_identities WHERE agent_id = $1 FOR UPDATE`, agentID).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAgentIdentityNotFound
		}
		return err
	}
	if current == "revoked" && status != "revoked" {
		return ErrAgentStatusTransition
	}
	if status == "active" && current != "active" {
		var hasActiveCertificate bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM agent_certificates
				WHERE agent_id = $1 AND status = 'active' AND not_after > now()
			)
		`, agentID).Scan(&hasActiveCertificate); err != nil {
			return err
		}
		if !hasActiveCertificate {
			return ErrAgentStatusTransition
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE agent_identities SET status = $2, updated_at = now() WHERE agent_id = $1`, agentID, status); err != nil {
		return err
	}
	if status == "revoked" {
		if _, err := tx.Exec(ctx, `
			UPDATE agent_certificates SET status = 'revoked', revoked_at = now()
			WHERE agent_id = $1 AND status = 'active'
		`, agentID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *AgentIdentityStore) ReportDevice(ctx context.Context, agentID string, info AgentDeviceInfo) error {
	capabilities, err := json.Marshal(info.Capabilities)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_identities SET
			device_driver = $2, device_hardware = $3, device_manufacturer = $4,
			device_product = $5, device_serial = $6, device_rx = $7, device_tx = $8,
			device_reported_at = now(), device_capabilities = $9::jsonb, last_seen_at = now(), updated_at = now()
		WHERE agent_id = $1 AND status = 'active'
	`, agentID, info.Driver, info.Hardware, info.Manufacturer, info.Product, info.Serial, info.RX, info.TX, string(capabilities))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentIdentityNotFound
	}
	return nil
}

func (s *AgentIdentityStore) ReportDeviceInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string, info AgentDeviceInfo) error {
	var previousJSON []byte
	var previous AgentDeviceCapabilities
	if err := s.pool.QueryRow(ctx, `SELECT device_capabilities FROM agent_identities WHERE agent_id=$1 AND organization_id=$2 AND status='active'`, agentID, organizationID).Scan(&previousJSON); err == nil {
		_ = json.Unmarshal(previousJSON, &previous)
	}
	applyIntegrityBaseline(previous, &info.Capabilities)
	capabilities, err := json.Marshal(info.Capabilities)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_identities SET
			device_driver = $3, device_hardware = $4, device_manufacturer = $5,
			device_product = $6, device_serial = $7, device_rx = $8, device_tx = $9,
			device_reported_at = now(), device_capabilities = $10::jsonb, last_seen_at = now(), updated_at = now()
		WHERE agent_id = $1 AND organization_id = $2 AND status = 'active'
	`, agentID, organizationID, info.Driver, info.Hardware, info.Manufacturer, info.Product, info.Serial, info.RX, info.TX, string(capabilities))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentIdentityNotFound
	}
	return nil
}

func (s *AgentIdentityStore) SetStation(ctx context.Context, agentID string, station AgentStation) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_identities
		SET station_latitude_deg = $2, station_longitude_deg = $3, station_altitude_m = $4, updated_at = now()
		WHERE agent_id = $1
	`, agentID, station.LatitudeDeg, station.LongitudeDeg, station.AltitudeM)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentIdentityNotFound
	}
	return nil
}

func (s *AgentIdentityStore) Station(ctx context.Context, agentID string) (AgentStation, error) {
	var station AgentStation
	err := s.pool.QueryRow(ctx, `
		SELECT station_latitude_deg, station_longitude_deg, station_altitude_m
		FROM agent_identities
		WHERE agent_id = $1
		  AND station_latitude_deg IS NOT NULL
		  AND station_longitude_deg IS NOT NULL
		  AND station_altitude_m IS NOT NULL
	`, agentID).Scan(&station.LatitudeDeg, &station.LongitudeDeg, &station.AltitudeM)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AgentStation{}, ErrAgentStationNotConfigured
		}
		return AgentStation{}, err
	}
	return station, nil
}

func (s *AgentIdentityStore) Status(ctx context.Context, agentID string) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `SELECT status FROM agent_identities WHERE agent_id = $1`, agentID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAgentIdentityNotFound
		}
		return "", err
	}
	return status, nil
}

func (s *AgentIdentityStore) ExistsInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM agent_identities
			WHERE agent_id=$1 AND organization_id=$2
		)
	`, agentID, organizationID).Scan(&ok)
	return ok, err
}

func (s *AgentIdentityStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID) ([]AgentIdentity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT agent_id, organization_id, status, first_seen_at, last_seen_at,
		       coalesce(agent_version,''), coalesce(firmware_version,''),
		       station_latitude_deg, station_longitude_deg, station_altitude_m,
		       device_driver, device_hardware, device_manufacturer, device_product,
		       device_serial, device_rx, device_tx, device_reported_at, device_capabilities
		FROM agent_identities
		WHERE organization_id=$1
		ORDER BY agent_id
	`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AgentIdentity{}
	for rows.Next() {
		var item AgentIdentity
		var lat, lon, alt *float64
		var driver, hardwareName, manufacturer, product, serial *string
		var rx, tx *bool
		var reportedAt *time.Time
		var capabilitiesJSON []byte
		if err := rows.Scan(&item.AgentID, &item.OrganizationID, &item.Status, &item.FirstSeenAt, &item.LastSeenAt,
			&item.AgentVersion, &item.FirmwareVersion, &lat, &lon, &alt,
			&driver, &hardwareName, &manufacturer, &product, &serial, &rx, &tx, &reportedAt, &capabilitiesJSON); err != nil {
			return nil, err
		}
		if lat != nil && lon != nil && alt != nil {
			item.Station = &AgentStation{LatitudeDeg: *lat, LongitudeDeg: *lon, AltitudeM: *alt}
		}
		if reportedAt != nil {
			item.Device = &AgentDeviceInfo{
				Driver: deref(driver), Hardware: deref(hardwareName), Manufacturer: deref(manufacturer), Product: deref(product),
				Serial: deref(serial), RX: rx != nil && *rx, TX: tx != nil && *tx, ReportedAt: *reportedAt,
			}
			_ = json.Unmarshal(capabilitiesJSON, &item.Device.Capabilities)
			finalizeAgentDeviceInfo(item.Device, time.Now().UTC())
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *AgentIdentityStore) CreateEnrollmentTokenInOrganization(ctx context.Context, organizationID uuid.UUID, agentID, tokenHash string, expiresAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `
		INSERT INTO agent_identities (agent_id, organization_id, status)
		VALUES ($1,$2,'pending')
		ON CONFLICT (agent_id) DO UPDATE SET updated_at=now()
		RETURNING status
	`, agentID, organizationID).Scan(&status)
	if err != nil {
		return err
	}

	var owner uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT organization_id FROM agent_identities WHERE agent_id=$1`, agentID).Scan(&owner); err != nil {
		return err
	}
	if owner != organizationID {
		return ErrAgentIdentityNotFound
	}
	if status == "revoked" || status == "quarantined" {
		return ErrAgentIdentityInactive
	}
	if status == "active" {
		var active bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM agent_certificates WHERE agent_id=$1 AND organization_id=$2 AND status='active' AND not_after>now())`, agentID, organizationID).Scan(&active); err != nil {
			return err
		}
		if active {
			return ErrAgentAlreadyEnrolled
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE agent_enrollment_tokens SET used_at=now() WHERE agent_id=$1 AND organization_id=$2 AND used_at IS NULL`, agentID, organizationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO agent_enrollment_tokens (agent_id, organization_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`,
		agentID, organizationID, tokenHash, expiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AgentIdentityStore) SetStationInOrganization(ctx context.Context, organizationID uuid.UUID, agentID string, station AgentStation) error {
	ok, err := s.ExistsInOrganization(ctx, organizationID, agentID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAgentIdentityNotFound
	}
	return s.SetStation(ctx, agentID, station)
}

func (s *AgentIdentityStore) SetStatusInOrganization(ctx context.Context, organizationID uuid.UUID, agentID, status string) error {
	ok, err := s.ExistsInOrganization(ctx, organizationID, agentID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAgentIdentityNotFound
	}
	return s.SetStatus(ctx, agentID, status)
}

func (s *AgentIdentityStore) OrganizationForAgent(ctx context.Context, agentID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM agent_identities WHERE agent_id=$1`, agentID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrAgentIdentityNotFound
	}
	return id, err
}
