-- v10.1: tenant boundary hardening for user administration.
-- Existing global uniqueness constraints remain intentionally unchanged in this phase
-- to preserve backward compatibility; authorization is scoped by organization_id.
CREATE INDEX IF NOT EXISTS users_org_enabled_role_idx
    ON users (organization_id, enabled, role)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS users_org_created_idx
    ON users (organization_id, created_at)
    WHERE deleted_at IS NULL;
