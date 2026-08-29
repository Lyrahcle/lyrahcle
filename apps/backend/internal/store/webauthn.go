package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebAuthnCredentialRecord struct {
	ID         uuid.UUID  `json:"id"`
	Label      string     `json:"label"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type WebAuthnStore struct {
	pool *pgxpool.Pool
}

func NewWebAuthnStore(pool *pgxpool.Pool) *WebAuthnStore {
	return &WebAuthnStore{pool: pool}
}

func (s *WebAuthnStore) Handle(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	var handle []byte
	err := s.pool.QueryRow(ctx, `SELECT webauthn_handle FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&handle)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return handle, err
}

func (s *WebAuthnStore) Credentials(ctx context.Context, userID uuid.UUID) ([]webauthn.Credential, error) {
	rows, err := s.pool.Query(ctx, `SELECT credential FROM webauthn_credentials WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]webauthn.Credential, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var credential webauthn.Credential
		if err := json.Unmarshal(raw, &credential); err != nil {
			return nil, err
		}
		out = append(out, credential)
	}
	return out, rows.Err()
}

func (s *WebAuthnStore) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM webauthn_credentials WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (s *WebAuthnStore) Add(ctx context.Context, userID uuid.UUID, label string, credential *webauthn.Credential) error {
	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO webauthn_credentials (user_id, credential_id, credential, label) VALUES ($1, $2, $3::jsonb, $4)`, userID, credential.ID, raw, label)
	return err
}

func (s *WebAuthnStore) UpdateCredential(ctx context.Context, userID uuid.UUID, credential *webauthn.Credential) error {
	raw, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE webauthn_credentials SET credential = $1::jsonb, last_used_at = now() WHERE user_id = $2 AND credential_id = $3`, raw, userID, credential.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("store: webauthn credential not found")
	}
	return nil
}

func (s *WebAuthnStore) List(ctx context.Context, userID uuid.UUID) ([]WebAuthnCredentialRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, label, created_at, last_used_at FROM webauthn_credentials WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WebAuthnCredentialRecord, 0)
	for rows.Next() {
		var item WebAuthnCredentialRecord
		if err := rows.Scan(&item.ID, &item.Label, &item.CreatedAt, &item.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *WebAuthnStore) Delete(ctx context.Context, userID, credentialID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM webauthn_credentials WHERE id = $1 AND user_id = $2`, credentialID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("store: webauthn credential not found")
	}
	return nil
}
