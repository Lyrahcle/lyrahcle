package store

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrOrganizationNotFound = errors.New("store: organization not found")
var ErrOrganizationSlugTaken = errors.New("store: organization slug already registered")

var DefaultOrganizationID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Organization struct {
	ID      uuid.UUID `json:"id"`
	Slug    string    `json:"slug"`
	Name    string    `json:"name"`
	Enabled bool      `json:"enabled"`
}

type OrganizationStore struct{ pool *pgxpool.Pool }

func NewOrganizationStore(pool *pgxpool.Pool) *OrganizationStore {
	return &OrganizationStore{pool: pool}
}

func (s *OrganizationStore) List(ctx context.Context) ([]Organization, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, slug, name, enabled FROM organizations ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Slug, &o.Name, &o.Enabled); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *OrganizationStore) GetByID(ctx context.Context, id uuid.UUID) (Organization, error) {
	var o Organization
	err := s.pool.QueryRow(ctx, `SELECT id, slug, name, enabled FROM organizations WHERE id = $1`, id).
		Scan(&o.ID, &o.Slug, &o.Name, &o.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrOrganizationNotFound
	}
	return o, err
}

func (s *OrganizationStore) Create(ctx context.Context, slug, name string) (Organization, error) {
	slug, name = strings.TrimSpace(strings.ToLower(slug)), strings.TrimSpace(name)
	var o Organization
	err := s.pool.QueryRow(ctx, `INSERT INTO organizations (slug, name) VALUES ($1,$2) RETURNING id, slug, name, enabled`, slug, name).
		Scan(&o.ID, &o.Slug, &o.Name, &o.Enabled)
	if err != nil {
		if isUniqueViolation(err) {
			return Organization{}, ErrOrganizationSlugTaken
		}
		return Organization{}, err
	}
	return o, nil
}
func (s *OrganizationStore) Update(ctx context.Context, id uuid.UUID, slug, name string) (Organization, error) {
	slug, name = strings.TrimSpace(strings.ToLower(slug)), strings.TrimSpace(name)
	var o Organization
	err := s.pool.QueryRow(ctx, `
		UPDATE organizations
		SET slug=$2, name=$3, updated_at=now()
		WHERE id=$1
		RETURNING id, slug, name, enabled
	`, id, slug, name).Scan(&o.ID, &o.Slug, &o.Name, &o.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return Organization{}, ErrOrganizationNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return Organization{}, ErrOrganizationSlugTaken
		}
		return Organization{}, err
	}
	return o, nil
}
