CREATE TABLE IF NOT EXISTS agent_identities (
    agent_id TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'quarantined', 'revoked', 'offline')),
    first_seen_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    agent_version TEXT NOT NULL DEFAULT '',
    firmware_version TEXT NOT NULL DEFAULT '',
    device_driver TEXT,
    device_hardware TEXT,
    device_manufacturer TEXT,
    device_product TEXT,
    device_serial TEXT,
    device_rx BOOLEAN,
    device_tx BOOLEAN,
    device_reported_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id TEXT NOT NULL REFERENCES agent_identities(agent_id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE CHECK (length(token_hash) = 64),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id TEXT NOT NULL REFERENCES agent_identities(agent_id) ON DELETE CASCADE,
    serial_number TEXT NOT NULL UNIQUE,
    fingerprint_sha256 TEXT NOT NULL UNIQUE CHECK (length(fingerprint_sha256) = 64),
    certificate_pem TEXT NOT NULL,
    not_before TIMESTAMPTZ NOT NULL,
    not_after TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'superseded', 'revoked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ,
    superseded_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_certificates_one_active
    ON agent_certificates(agent_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_agent_certificates_fingerprint_active
    ON agent_certificates(fingerprint_sha256)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_agent_enrollment_tokens_active
    ON agent_enrollment_tokens(token_hash, expires_at)
    WHERE used_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_identities_status_last_seen
    ON agent_identities(status, last_seen_at DESC);
