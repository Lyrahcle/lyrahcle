CREATE TABLE IF NOT EXISTS rf_readings (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id      TEXT NOT NULL,
    frequency_hz  BIGINT NOT NULL,
    power_dbm     DOUBLE PRECISION NOT NULL,
    captured_at   TIMESTAMPTZ NOT NULL,
    received_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_rf_readings_agent ON rf_readings(agent_id);
CREATE INDEX IF NOT EXISTS idx_rf_readings_captured ON rf_readings(captured_at);
