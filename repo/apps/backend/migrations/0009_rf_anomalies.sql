CREATE TABLE IF NOT EXISTS rf_anomalies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reading_id      UUID NOT NULL REFERENCES rf_readings(id) ON DELETE CASCADE,
    agent_id        TEXT NOT NULL,
    metric          TEXT NOT NULL,
    value           DOUBLE PRECISION NOT NULL,
    baseline_mean   DOUBLE PRECISION NOT NULL,
    baseline_stddev DOUBLE PRECISION NOT NULL,
    z_score         DOUBLE PRECISION NOT NULL,
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rf_anomalies_agent ON rf_anomalies(agent_id);
CREATE INDEX IF NOT EXISTS idx_rf_anomalies_detected ON rf_anomalies(detected_at);
