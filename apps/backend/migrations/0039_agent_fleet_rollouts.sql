CREATE TABLE IF NOT EXISTS agent_fleet_rollouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    frequency_hz BIGINT NOT NULL CHECK (frequency_hz BETWEEN 100000 AND 6000000000),
    capture_interval_seconds INTEGER NOT NULL CHECK (capture_interval_seconds BETWEEN 1 AND 3600),
    satellite_id UUID REFERENCES satellites(id) ON DELETE SET NULL,
    batch_size INTEGER NOT NULL DEFAULT 1 CHECK (batch_size BETWEEN 1 AND 1000),
    failure_threshold_percent INTEGER NOT NULL DEFAULT 20 CHECK (failure_threshold_percent BETWEEN 0 AND 100),
    soak_seconds INTEGER NOT NULL DEFAULT 300 CHECK (soak_seconds BETWEEN 0 AND 86400),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','running','paused','completed','rolled_back','failed')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_fleet_rollout_targets (
    rollout_id UUID NOT NULL REFERENCES agent_fleet_rollouts(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL REFERENCES agent_identities(agent_id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','issued','applied','rejected','rolled_back')),
    config_version BIGINT,
    previous_version BIGINT,
    issued_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (rollout_id, agent_id),
    UNIQUE (rollout_id, ordinal)
);

CREATE INDEX IF NOT EXISTS idx_agent_fleet_rollouts_org_created ON agent_fleet_rollouts(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_fleet_targets_rollout_status ON agent_fleet_rollout_targets(rollout_id, status, ordinal);
