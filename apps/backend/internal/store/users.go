package store

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("store: user not found")
var ErrEmailTaken = errors.New("store: email already registered")

var usernameSanitizer = regexp.MustCompile(`[^a-z0-9_.-]`)

type User struct {
	ID                 uuid.UUID `json:"id"`
	Email              string    `json:"email"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"`
	Role               string    `json:"role"`
	TOTPSecret         string    `json:"-"`
	TOTPEnabled        bool      `json:"totp_enabled"`
	Enabled            bool      `json:"enabled"`
	MustChangePassword bool      `json:"must_change_password"`
	AuthVersion        int64     `json:"-"`
}

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count)
	return count, err
}

func (s *UserStore) Create(ctx context.Context, email, passwordHash, role string) (User, error) {
	base := usernameSanitizer.ReplaceAllString(strings.ToLower(strings.SplitN(email, "@", 2)[0]), "")
	if base == "" {
		base = "user"
	}

	var u User
	username := base
	for attempt := 0; attempt < 5; attempt++ {
		err := s.pool.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, role)
			VALUES ($1, $2, $3, $4)
			RETURNING id, email, username, password_hash, role, coalesce(totp_secret, ''), totp_enabled, enabled, must_change_password, auth_version
		`, email, username, passwordHash, role).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.Enabled, &u.MustChangePassword, &u.AuthVersion)
		if err == nil {
			return u, nil
		}
		if isUniqueViolation(err) {
			if strings.Contains(err.Error(), "users_username_key") {
				username = base + "-" + uuid.NewString()[:6]
				continue
			}
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return User{}, errors.New("store: could not allocate a unique username")
}

func (s *UserStore) CreateProvisioned(ctx context.Context, email, passwordHash, role string) (User, error) {
	u, err := s.Create(ctx, email, passwordHash, role)
	if err != nil {
		return User{}, err
	}
	if _, err := s.pool.Exec(ctx, `UPDATE users SET must_change_password = true, auth_version = auth_version + 1 WHERE id = $1`, u.ID); err != nil {
		return User{}, err
	}
	u.MustChangePassword = true
	u.AuthVersion++
	return u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, role, coalesce(totp_secret, ''), totp_enabled, enabled, must_change_password, auth_version FROM users WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.Enabled, &u.MustChangePassword, &u.AuthVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (s *UserStore) GetByIdentifier(ctx context.Context, identifier string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, role, coalesce(totp_secret, ''), totp_enabled, enabled, must_change_password, auth_version FROM users
		WHERE (email = $1 OR username = $1) AND deleted_at IS NULL
	`, identifier).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.Enabled, &u.MustChangePassword, &u.AuthVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, username, password_hash, role, coalesce(totp_secret, ''), totp_enabled, enabled, must_change_password, auth_version FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.Enabled, &u.MustChangePassword, &u.AuthVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (s *UserStore) UpdatePassword(ctx context.Context, id uuid.UUID, newHash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $1, must_change_password = false, password_changed_at = now(), auth_version = auth_version + 1 WHERE id = $2 AND deleted_at IS NULL`, newHash, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *UserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, email, username, password_hash, role, coalesce(totp_secret, ''), totp_enabled, enabled, must_change_password, auth_version FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.Enabled, &u.MustChangePassword, &u.AuthVersion); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *UserStore) UpdateRole(ctx context.Context, id uuid.UUID, role string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		UPDATE users SET role = $1, auth_version = auth_version + 1 WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, email, username, password_hash, role, coalesce(totp_secret, ''), totp_enabled, enabled, must_change_password, auth_version
	`, role, id).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.Enabled, &u.MustChangePassword, &u.AuthVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (s *UserStore) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE role = 'admin' AND enabled = true AND deleted_at IS NULL`).Scan(&count)
	return count, err
}

func (s *UserStore) SetPendingTOTPSecret(ctx context.Context, id uuid.UUID, secret string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET totp_secret = $1, totp_enabled = false WHERE id = $2`, secret, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *UserStore) EnableTOTP(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET totp_enabled = true, auth_version = auth_version + 1 WHERE id = $1 AND totp_secret IS NOT NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *UserStore) DisableTOTP(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET totp_secret = NULL, totp_enabled = false, auth_version = auth_version + 1 WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *UserStore) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		UPDATE users SET enabled = $1, auth_version = auth_version + 1 WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, email, username, password_hash, role, coalesce(totp_secret, ''), totp_enabled, enabled, must_change_password, auth_version
	`, enabled, id).Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled, &u.Enabled, &u.MustChangePassword, &u.AuthVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return u, err
}

func (s *UserStore) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET enabled = false, deleted_at = now(), email = 'deleted-' || id::text || '@invalid.local', username = 'deleted-' || substr(id::text, 1, 12), password_hash = '!deleted!', totp_secret = NULL, totp_enabled = false, must_change_password = false, auth_version = auth_version + 1 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
