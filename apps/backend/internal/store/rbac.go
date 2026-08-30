package store

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrOrganizationRoleNotFound = errors.New("store: organization role not found")
var ErrOrganizationRoleNameTaken = errors.New("store: organization role name taken")

type Permission struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type OrganizationRole struct {
	ID             uuid.UUID    `json:"id"`
	OrganizationID uuid.UUID    `json:"organization_id"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	System         bool         `json:"system"`
	Permissions    []Permission `json:"permissions,omitempty"`
}

type RBACStore struct {
	pool *pgxpool.Pool
}

func NewRBACStore(pool *pgxpool.Pool) *RBACStore { return &RBACStore{pool: pool} }

func (s *RBACStore) HasPermission(ctx context.Context, userID, organizationID uuid.UUID, permission string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM organization_membership_roles umr
			JOIN organization_role_permissions rp ON rp.organization_role_id=umr.organization_role_id
			WHERE umr.user_id=$1 AND umr.organization_id=$2 AND rp.permission_key=$3
		)
	`, userID, organizationID, permission).Scan(&ok)
	return ok, err
}

func (s *RBACStore) EffectivePermissions(ctx context.Context, userID, organizationID uuid.UUID) ([]Permission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT p.key,p.description
		FROM organization_membership_roles umr
		JOIN organization_role_permissions rp ON rp.organization_role_id=umr.organization_role_id
		JOIN permissions p ON p.key=rp.permission_key
		WHERE umr.user_id=$1 AND umr.organization_id=$2
		ORDER BY p.key
	`, userID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Permission{}
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Key, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *RBACStore) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := s.pool.Query(ctx, `SELECT key,description FROM permissions ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Permission{}
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Key, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *RBACStore) ListRoles(ctx context.Context, organizationID uuid.UUID) ([]OrganizationRole, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id,organization_id,name,description,system
		FROM organization_roles WHERE organization_id=$1 ORDER BY system DESC,name
	`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrganizationRole{}
	for rows.Next() {
		var r OrganizationRole
		if err := rows.Scan(&r.ID, &r.OrganizationID, &r.Name, &r.Description, &r.System); err != nil {
			return nil, err
		}
		perms, err := s.permissionsForRole(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		r.Permissions = perms
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *RBACStore) permissionsForRole(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.key,p.description
		FROM organization_role_permissions rp
		JOIN permissions p ON p.key=rp.permission_key
		WHERE rp.organization_role_id=$1 ORDER BY p.key
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Permission{}
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Key, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *RBACStore) CreateRole(ctx context.Context, organizationID uuid.UUID, name, description string, permissionKeys []string) (OrganizationRole, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OrganizationRole{}, err
	}
	defer tx.Rollback(ctx)

	var r OrganizationRole
	err = tx.QueryRow(ctx, `
		INSERT INTO organization_roles (organization_id,name,description,system)
		VALUES ($1,$2,$3,false)
		RETURNING id,organization_id,name,description,system
	`, organizationID, name, strings.TrimSpace(description)).Scan(&r.ID, &r.OrganizationID, &r.Name, &r.Description, &r.System)
	if err != nil {
		if strings.Contains(err.Error(), "organization_roles_organization_id_name_key") {
			return OrganizationRole{}, ErrOrganizationRoleNameTaken
		}
		return OrganizationRole{}, err
	}
	for _, key := range permissionKeys {
		tag, err := tx.Exec(ctx, `
			INSERT INTO organization_role_permissions (organization_role_id,permission_key)
			SELECT $1,key FROM permissions WHERE key=$2
			ON CONFLICT DO NOTHING
		`, r.ID, key)
		if err != nil {
			return OrganizationRole{}, err
		}
		if tag.RowsAffected() == 0 {
			return OrganizationRole{}, errors.New("store: unknown permission " + key)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return OrganizationRole{}, err
	}
	r.Permissions, err = s.permissionsForRole(ctx, r.ID)
	return r, err
}

func (s *RBACStore) AssignRole(ctx context.Context, organizationID, userID, roleID uuid.UUID) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM organization_roles
			WHERE id=$1 AND organization_id=$2
		)
	`, roleID, organizationID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrOrganizationRoleNotFound
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO organization_membership_roles (user_id,organization_id,organization_role_id)
		SELECT $1,$2,$3
		WHERE EXISTS(SELECT 1 FROM users WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL)
		ON CONFLICT DO NOTHING
	`, userID, organizationID, roleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var userExists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND organization_id=$2 AND deleted_at IS NULL)`, userID, organizationID).Scan(&userExists); err != nil {
			return err
		}
		if !userExists {
			return ErrUserNotFound
		}
	}
	return nil
}

func (s *RBACStore) RemoveRole(ctx context.Context, organizationID, userID, roleID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM organization_membership_roles
		WHERE user_id=$1 AND organization_id=$2 AND organization_role_id=$3
	`, userID, organizationID, roleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
