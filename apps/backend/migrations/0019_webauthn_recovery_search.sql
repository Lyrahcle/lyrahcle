ALTER TABLE users ADD COLUMN IF NOT EXISTS webauthn_handle BYTEA;
UPDATE users SET webauthn_handle = gen_random_bytes(32) WHERE webauthn_handle IS NULL;
ALTER TABLE users ALTER COLUMN webauthn_handle SET DEFAULT gen_random_bytes(32);
ALTER TABLE users ALTER COLUMN webauthn_handle SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_webauthn_handle ON users (webauthn_handle) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    credential JSONB NOT NULL,
    label VARCHAR(80) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user ON webauthn_credentials (user_id, created_at);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_password_reset_active ON password_reset_tokens (user_id, expires_at) WHERE used_at IS NULL;

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_satellites_name_trgm ON satellites USING gin (lower(name) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_assets_value_trgm ON assets USING gin (lower(value) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_assets_kind_trgm ON assets USING gin (lower(kind) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_incidents_reference_trgm ON incidents USING gin (lower(reference) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_incidents_title_trgm ON incidents USING gin (lower(title) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_space_cves_cve_id_trgm ON space_cves USING gin (lower(cve_id) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_space_cves_description_trgm ON space_cves USING gin (lower(description) gin_trgm_ops);
