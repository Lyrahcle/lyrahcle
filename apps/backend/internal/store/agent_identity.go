package store

import (
	"context"
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

type AgentIdentityStore struct {
	pool *pgxpool.Pool
}

type AgentIdentity struct {
	AgentID         string            `json:"agent_id"`
	Status          string            `json:"status"`
	FirstSeenAt     *time.Time        `json:"first_seen_at"`
	LastSeenAt      *time.Time        `json:"last_seen_at"`
	AgentVersion    string            `json:"agent_version"`
	FirmwareVersion string            `json:"firmware_version"`
	Certificate     *AgentCertificate `json:"certificate,omitempty"`
	Device          *AgentDeviceInfo  `json:"device,omitempty"`
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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `
		INSERT INTO agent_identities (agent_id, status)
		VALUES ($1, 'pending')
		ON CONFLICT (agent_id) DO UPDATE SET updated_at = now()
		RETURNING status
	`, agentID).Scan(&status)
	if err != nil {
		return err
	}
	if status == "revoked" || status == "quarantined" {
		return ErrAgentIdentityInactive
	}
	if status == "active" {
		var hasValidCertificate bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM agent_certificates
				WHERE agent_id = $1 AND status = 'active' AND not_after > now()
			)
		`, agentID).Scan(&hasValidCertificate); err != nil {
			return err
		}
		if hasValidCertificate {
			return ErrAgentAlreadyEnrolled
		}
		if _, err := tx.Exec(ctx, `
			UPDATE agent_certificates
			SET status = 'superseded', superseded_at = now()
			WHERE agent_id = $1 AND status = 'active'
		`, agentID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE agent_identities SET status = 'pending', updated_at = now() WHERE agent_id = $1`, agentID); err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE agent_enrollment_tokens
		SET used_at = now()
		WHERE agent_id = $1 AND used_at IS NULL
	`, agentID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO agent_enrollment_tokens (agent_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, agentID, tokenHash, expiresAt)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
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
	var status string
	err = tx.QueryRow(ctx, `
		SELECT t.agent_id, i.status
		FROM agent_enrollment_tokens t
		JOIN agent_identities i ON i.agent_id = t.agent_id
		WHERE t.token_hash = $1 AND t.used_at IS NULL AND t.expires_at > now()
		FOR UPDATE OF t, i
	`, tokenHash).Scan(&agentID, &status)
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
	if err := insertAgentCertificate(ctx, tx, cert); err != nil {
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

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM agent_identities WHERE agent_id = $1 FOR UPDATE`, agentID).Scan(&status)
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
	if err := insertAgentCertificate(ctx, tx, cert); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE agent_identities SET updated_at = now() WHERE agent_id = $1`, agentID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertAgentCertificate(ctx context.Context, tx pgx.Tx, cert AgentCertificate) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO agent_certificates (
			agent_id, serial_number, fingerprint_sha256, certificate_pem, not_before, not_after, status
		) VALUES ($1, $2, $3, $4, $5, $6, 'active')
	`, cert.AgentID, cert.SerialNumber, cert.FingerprintSHA256, cert.CertificatePEM, cert.NotBefore, cert.NotAfter)
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
	_, _ = s.pool.Exec(ctx, `
		UPDATE agent_certificates SET last_seen_at = now()
		WHERE fingerprint_sha256 = $1 AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
	`, fingerprint)
	_, _ = s.pool.Exec(ctx, `
		UPDATE agent_identities SET last_seen_at = now(), updated_at = now()
		WHERE agent_id = $1 AND (last_seen_at IS NULL OR last_seen_at < now() - interval '1 minute')
	`, agentID)
	return agentID, nil
}

func (s *AgentIdentityStore) List(ctx context.Context) ([]AgentIdentity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			i.agent_id, i.status, i.first_seen_at, i.last_seen_at, i.agent_version, i.firmware_version,
			i.device_driver, i.device_hardware, i.device_manufacturer, i.device_product,
			i.device_serial, i.device_rx, i.device_tx, i.device_reported_at,
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
		var certID *uuid.UUID
		var certSerial, fingerprint, certStatus *string
		var notBefore, notAfter, certLastSeen *time.Time
		if err := rows.Scan(
			&item.AgentID, &item.Status, &item.FirstSeenAt, &item.LastSeenAt, &item.AgentVersion, &item.FirmwareVersion,
			&driver, &hardware, &manufacturer, &product, &serial, &rx, &tx, &reportedAt,
			&certID, &certSerial, &fingerprint, &notBefore, &notAfter, &certStatus, &certLastSeen,
		); err != nil {
			return nil, err
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
	tag, err := s.pool.Exec(ctx, `
		UPDATE agent_identities SET
			device_driver = $2, device_hardware = $3, device_manufacturer = $4,
			device_product = $5, device_serial = $6, device_rx = $7, device_tx = $8,
			device_reported_at = now(), last_seen_at = now(), updated_at = now()
		WHERE agent_id = $1 AND status = 'active'
	`, agentID, info.Driver, info.Hardware, info.Manufacturer, info.Product, info.Serial, info.RX, info.TX)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAgentIdentityNotFound
	}
	return nil
}
