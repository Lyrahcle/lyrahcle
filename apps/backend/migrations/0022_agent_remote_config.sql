CREATE TABLE IF NOT EXISTS agent_config_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id TEXT NOT NULL REFERENCES agent_identities(agent_id) ON DELETE CASCADE,
    version BIGINT NOT NULL CHECK (version > 0),
    frequency_hz BIGINT NOT NULL CHECK (frequency_hz BETWEEN 100000 AND 6000000000),
    capture_interval_seconds INTEGER NOT NULL CHECK (capture_interval_seconds BETWEEN 1 AND 3600),
    payload BYTEA NOT NULL,
    signature BYTEA NOT NULL CHECK (octet_length(signature) = 64),
    checksum_sha256 TEXT NOT NULL CHECK (length(checksum_sha256) = 64),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (agent_id, version)
);

CREATE TABLE IF NOT EXISTS agent_config_state (
    agent_id TEXT PRIMARY KEY REFERENCES agent_identities(agent_id) ON DELETE CASCADE,
    desired_version BIGINT NOT NULL CHECK (desired_version > 0),
    acknowledged_version BIGINT,
    ack_status TEXT NOT NULL DEFAULT 'pending' CHECK (ack_status IN ('pending', 'applied', 'rejected')),
    ack_message TEXT NOT NULL DEFAULT '' CHECK (length(ack_message) <= 256),
    acked_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_config_acks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id TEXT NOT NULL REFERENCES agent_identities(agent_id) ON DELETE CASCADE,
    version BIGINT NOT NULL CHECK (version > 0),
    status TEXT NOT NULL CHECK (status IN ('applied', 'rejected')),
    message TEXT NOT NULL DEFAULT '' CHECK (length(message) <= 256),
    checksum_sha256 TEXT NOT NULL CHECK (length(checksum_sha256) = 64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (agent_id, version)
);

CREATE INDEX IF NOT EXISTS idx_agent_config_versions_agent_created
    ON agent_config_versions(agent_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_config_state_ack
    ON agent_config_state(ack_status, updated_at DESC);
