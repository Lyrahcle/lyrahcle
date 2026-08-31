-- v13.10: richer SDR hardware capability reports for Fleet readiness.
ALTER TABLE agent_identities
    ADD COLUMN IF NOT EXISTS device_capabilities JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE agent_keys
    ADD COLUMN IF NOT EXISTS device_capabilities JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS agent_identities_device_capabilities_gin
    ON agent_identities USING GIN (device_capabilities);
