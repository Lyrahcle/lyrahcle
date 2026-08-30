-- v10.3: support tenant-scoped operational handlers.
-- Assets must be unique inside a tenant, not globally across tenants.
ALTER TABLE assets DROP CONSTRAINT IF EXISTS assets_program_id_kind_value_key;
CREATE UNIQUE INDEX IF NOT EXISTS assets_org_program_kind_value_uidx
    ON assets (organization_id, program_id, kind, value);

CREATE INDEX IF NOT EXISTS incidents_org_status_created_idx
    ON incidents (organization_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS incidents_org_severity_created_idx
    ON incidents (organization_id, severity, created_at DESC);
