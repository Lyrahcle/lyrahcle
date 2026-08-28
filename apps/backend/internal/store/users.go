package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("store: user not found")
var ErrEmailTaken = errors.New("store: email already registered")

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
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
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, role
	`, email, passwordHash, role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
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
		SELECT id, email, password_hash, role FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return u, nil
}

func (s *UserStore) UpdatePassword(ctx context.Context, id uuid.UUID, newHash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, newHash, id)
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
		SELECT id, email, password_hash, role FROM users ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *UserStore) UpdateRole(ctx context.Context, id uuid.UUID, role string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		UPDATE users SET role = $1 WHERE id = $2
		RETURNING id, email, password_hash, role
	`, role, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
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
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE role = 'admin'`).Scan(&count)
	return count, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
