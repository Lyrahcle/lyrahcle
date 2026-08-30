-- v10.5: tenant-local operational data.
-- Satellites and space_cves intentionally remain shared reference catalogs.

ALTER TABLE programs ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE rf_readings ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE rf_anomalies ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE orbital_events ADD COLUMN IF NOT EXISTS organization_id UUID;

UPDATE programs SET organization_id='00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;

UPDATE rf_readings r
SET organization_id=i.organization_id
FROM agent_identities i
WHERE i.agent_id=r.agent_id AND r.organization_id IS NULL;
UPDATE rf_readings SET organization_id='00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;

UPDATE rf_anomalies a
SET organization_id=r.organization_id
FROM rf_readings r
WHERE r.id=a.reading_id AND a.organization_id IS NULL;
UPDATE rf_anomalies SET organization_id='00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;

UPDATE orbital_events e
SET organization_id=u.organization_id
FROM users u
WHERE u.id=e.created_by AND e.organization_id IS NULL;
UPDATE orbital_events SET organization_id='00000000-0000-0000-0000-000000000001' WHERE organization_id IS NULL;

ALTER TABLE programs ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE rf_readings ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE rf_anomalies ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE orbital_events ALTER COLUMN organization_id SET DEFAULT '00000000-0000-0000-0000-000000000001';

ALTER TABLE programs ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE rf_readings ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE rf_anomalies ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE orbital_events ALTER COLUMN organization_id SET NOT NULL;

ALTER TABLE programs DROP CONSTRAINT IF EXISTS programs_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS programs_org_name_uidx ON programs (organization_id, lower(name));

CREATE INDEX IF NOT EXISTS rf_readings_org_captured_idx ON rf_readings (organization_id, captured_at DESC);
CREATE INDEX IF NOT EXISTS rf_anomalies_org_detected_idx ON rf_anomalies (organization_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS orbital_events_org_satellite_time_idx ON orbital_events (organization_id, satellite_id, event_time DESC);
