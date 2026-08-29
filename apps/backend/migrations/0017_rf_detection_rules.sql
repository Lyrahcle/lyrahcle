CREATE TABLE IF NOT EXISTS rf_detection_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT true,
    metric TEXT NOT NULL,
    operator TEXT NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    severity TEXT NOT NULL,
    cooldown_seconds INTEGER NOT NULL DEFAULT 120,
    auto_incident BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rf_detection_rules_name_length_check CHECK (char_length(name) BETWEEN 1 AND 120),
    CONSTRAINT rf_detection_rules_description_length_check CHECK (char_length(description) <= 1000),
    CONSTRAINT rf_detection_rules_metric_check CHECK (metric IN ('any', 'power_dbm', 'frequency_hz', 'power_dbm_trend', 'frequency_hz_trend')),
    CONSTRAINT rf_detection_rules_operator_check CHECK (operator IN ('abs_z_gte', 'z_gte', 'z_lte')),
    CONSTRAINT rf_detection_rules_threshold_check CHECK (threshold > 0 AND threshold <= 100),
    CONSTRAINT rf_detection_rules_severity_check CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    CONSTRAINT rf_detection_rules_cooldown_check CHECK (cooldown_seconds BETWEEN 0 AND 86400)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_rf_detection_rules_name
    ON rf_detection_rules(lower(name));

CREATE INDEX IF NOT EXISTS idx_rf_detection_rules_enabled
    ON rf_detection_rules(enabled, updated_at DESC);

CREATE TABLE IF NOT EXISTS rf_detection_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES rf_detection_rules(id) ON DELETE RESTRICT,
    anomaly_id UUID NOT NULL REFERENCES rf_anomalies(id) ON DELETE CASCADE,
    incident_id UUID REFERENCES incidents(id) ON DELETE SET NULL,
    agent_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    z_score DOUBLE PRECISION NOT NULL,
    matched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rf_detection_matches_agent_length_check CHECK (char_length(agent_id) BETWEEN 1 AND 128),
    CONSTRAINT rf_detection_matches_metric_check CHECK (metric IN ('power_dbm', 'frequency_hz', 'power_dbm_trend', 'frequency_hz_trend')),
    UNIQUE (rule_id, anomaly_id)
);

CREATE INDEX IF NOT EXISTS idx_rf_detection_matches_rule_agent
    ON rf_detection_matches(rule_id, agent_id, matched_at DESC);

CREATE INDEX IF NOT EXISTS idx_rf_detection_matches_anomaly
    ON rf_detection_matches(anomaly_id, matched_at ASC);

ALTER TABLE incidents
    DROP CONSTRAINT IF EXISTS incidents_source_type_check;

ALTER TABLE incidents
    ADD CONSTRAINT incidents_source_type_check
    CHECK (source_type IN ('manual', 'rf_anomaly', 'detection_rule'));

ALTER TABLE incident_evidence
    DROP CONSTRAINT IF EXISTS incident_evidence_type_check;

ALTER TABLE incident_evidence
    ADD CONSTRAINT incident_evidence_type_check
    CHECK (evidence_type IN ('rf_anomaly', 'detection_rule', 'request', 'cve', 'agent', 'satellite', 'external'));

INSERT INTO rf_detection_rules (name, description, metric, operator, threshold, severity, cooldown_seconds, auto_incident)
VALUES
    ('High RF deviation', 'Correlates high-confidence RF deviations across supported metrics.', 'any', 'abs_z_gte', 6, 'high', 120, true),
    ('Critical RF deviation', 'Escalates extreme RF deviations to critical incidents.', 'any', 'abs_z_gte', 8, 'critical', 120, true)
ON CONFLICT DO NOTHING;
