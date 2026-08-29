CREATE INDEX IF NOT EXISTS idx_rf_readings_agent_captured
    ON rf_readings(agent_id, captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_rf_anomalies_agent_detected
    ON rf_anomalies(agent_id, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_rf_anomalies_metric_detected
    ON rf_anomalies(metric, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_log_action_occurred
    ON audit_log(action, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_satellites_updated_at
    ON satellites(updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires
    ON password_reset_tokens(expires_at)
    WHERE used_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_last_used
    ON webauthn_credentials(user_id, last_used_at DESC NULLS LAST);
