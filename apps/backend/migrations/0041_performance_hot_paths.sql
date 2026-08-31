-- v14.6.4: tenant-aware hot-path indexes for operator-facing reads.
-- These complement the broad tenant indexes added in v10.x with the actual
-- secondary predicates used by RF correlation, incident filters, and audit
-- evidence lookups.

CREATE INDEX IF NOT EXISTS rf_readings_org_agent_captured_idx
    ON rf_readings (organization_id, agent_id, captured_at DESC);

CREATE INDEX IF NOT EXISTS audit_log_actor_occurred_idx
    ON audit_log (actor_type, actor_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS incidents_org_agent_created_idx
    ON incidents (organization_id, agent_id, created_at DESC)
    WHERE agent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS incidents_org_assigned_created_idx
    ON incidents (organization_id, assigned_to, created_at DESC)
    WHERE assigned_to IS NOT NULL;

CREATE INDEX IF NOT EXISTS incidents_org_satellite_created_idx
    ON incidents (organization_id, satellite_id, created_at DESC)
    WHERE satellite_id IS NOT NULL;
