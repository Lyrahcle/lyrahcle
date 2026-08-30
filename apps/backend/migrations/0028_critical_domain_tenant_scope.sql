-- v10.2: organization scope for critical operational domains.
-- Backfill keeps existing single-tenant installations compatible.

ALTER TABLE agent_identities ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE assets ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE incidents ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS organization_id UUID;

UPDATE agent_identities SET organization_id = '00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;
UPDATE assets SET organization_id = '00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;
UPDATE incidents SET organization_id = '00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;
UPDATE audit_log SET organization_id = '00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;

ALTER TABLE agent_identities ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE assets ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE incidents ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE audit_log ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';

ALTER TABLE agent_identities ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE assets ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE incidents ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE audit_log ALTER COLUMN organization_id SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='agent_identities_organization_id_fkey') THEN
    ALTER TABLE agent_identities ADD CONSTRAINT agent_identities_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='assets_organization_id_fkey') THEN
    ALTER TABLE assets ADD CONSTRAINT assets_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='incidents_organization_id_fkey') THEN
    ALTER TABLE incidents ADD CONSTRAINT incidents_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='audit_log_organization_id_fkey') THEN
    ALTER TABLE audit_log ADD CONSTRAINT audit_log_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS agent_identities_org_seen_idx ON agent_identities (organization_id, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS assets_org_created_idx ON assets (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS incidents_org_created_idx ON incidents (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_org_created_idx ON audit_log (organization_id, created_at DESC);
