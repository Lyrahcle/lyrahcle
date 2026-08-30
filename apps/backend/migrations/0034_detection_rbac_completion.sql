-- v11.7: tenant-scope RF detection policy and complete operational permissions.

INSERT INTO permissions (key, description) VALUES
('detections.read','Read organization RF detection rules'),
('detections.manage','Create/update organization RF detection rules'),
('alerts.read','Read organization operational alerts'),
('alerts.manage','Acknowledge/manage organization operational alerts')
ON CONFLICT (key) DO NOTHING;

-- Extend compatibility presets without changing custom roles.
INSERT INTO organization_role_permissions (organization_role_id, permission_key)
SELECT r.id, p.key
FROM organization_roles r
JOIN permissions p ON (
    (r.name IN ('viewer','auditor','analyst','operator','admin') AND p.key IN ('detections.read','alerts.read'))
    OR (r.name IN ('operator','admin') AND p.key IN ('detections.manage','alerts.manage'))
)
WHERE r.system = true
ON CONFLICT DO NOTHING;

-- Ensure future organizations receive the v11.7 permissions as part of compatibility seeding.
CREATE OR REPLACE FUNCTION lyrahcle_seed_organization_rbac(target_organization_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO organization_roles (organization_id, name, description, system)
    VALUES
      (target_organization_id, 'viewer', 'Compatibility role for viewer', true),
      (target_organization_id, 'auditor', 'Compatibility role for auditor', true),
      (target_organization_id, 'analyst', 'Compatibility role for analyst', true),
      (target_organization_id, 'operator', 'Compatibility role for operator', true),
      (target_organization_id, 'admin', 'Compatibility role for admin', true)
    ON CONFLICT (organization_id, name) DO NOTHING;

    INSERT INTO organization_role_permissions (organization_role_id, permission_key)
    SELECT r.id, p.key
    FROM organization_roles r
    JOIN permissions p ON (
        r.name = 'admin'
        OR (r.name = 'viewer' AND p.key = ANY(ARRAY[
            'assets.read','incidents.read','programs.read','rf.read','orbital.read','agents.read',
            'detections.read','alerts.read'
        ]))
        OR (r.name = 'auditor' AND p.key = ANY(ARRAY[
            'assets.read','incidents.read','programs.read','rf.read','orbital.read','agents.read','audit.read',
            'detections.read','alerts.read'
        ]))
        OR (r.name = 'analyst' AND p.key = ANY(ARRAY[
            'assets.read','incidents.read','incidents.manage','programs.read','rf.read','orbital.read','agents.read','audit.read',
            'detections.read','alerts.read'
        ]))
        OR (r.name = 'operator' AND p.key = ANY(ARRAY[
            'assets.read','assets.manage','incidents.read','incidents.manage','programs.read','programs.manage',
            'rf.read','rf.ingest','orbital.read','orbital.manage','agents.read','agents.manage','audit.read',
            'detections.read','detections.manage','alerts.read','alerts.manage'
        ]))
    )
    WHERE r.organization_id = target_organization_id
      AND r.system = true
      AND r.name IN ('viewer','auditor','analyst','operator','admin')
    ON CONFLICT DO NOTHING;
END;
$$;

ALTER TABLE rf_detection_rules ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
UPDATE rf_detection_rules
SET organization_id = '00000000-0000-0000-0000-000000000001'
WHERE organization_id IS NULL;
ALTER TABLE rf_detection_rules ALTER COLUMN organization_id SET NOT NULL;

DROP INDEX IF EXISTS idx_rf_detection_rules_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_rf_detection_rules_org_name
    ON rf_detection_rules(organization_id, lower(name));
DROP INDEX IF EXISTS idx_rf_detection_rules_enabled;
CREATE INDEX IF NOT EXISTS idx_rf_detection_rules_org_enabled
    ON rf_detection_rules(organization_id, enabled, updated_at DESC);

ALTER TABLE rf_detection_matches ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
UPDATE rf_detection_matches m
SET organization_id = a.organization_id
FROM rf_anomalies a
WHERE a.id = m.anomaly_id AND m.organization_id IS NULL;
UPDATE rf_detection_matches
SET organization_id = '00000000-0000-0000-0000-000000000001'
WHERE organization_id IS NULL;
ALTER TABLE rf_detection_matches ALTER COLUMN organization_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rf_detection_matches_org_rule_agent
    ON rf_detection_matches(organization_id, rule_id, agent_id, matched_at DESC);

-- Seed the built-in detection policy into every existing organization. The original
-- pre-tenant rows remain in the default organization and are not duplicated there.
INSERT INTO rf_detection_rules (
    organization_id, name, description, enabled, metric, operator, threshold, severity,
    cooldown_seconds, auto_incident
)
SELECT o.id, seed.name, seed.description, true, 'any', 'abs_z_gte', seed.threshold, seed.severity, 120, true
FROM organizations o
CROSS JOIN (VALUES
    ('High RF deviation', 'Correlates high-confidence RF deviations across supported metrics.', 6::double precision, 'high'),
    ('Critical RF deviation', 'Escalates extreme RF deviations to critical incidents.', 8::double precision, 'critical')
) AS seed(name, description, threshold, severity)
WHERE o.id <> '00000000-0000-0000-0000-000000000001'
ON CONFLICT DO NOTHING;

CREATE OR REPLACE FUNCTION lyrahcle_seed_detection_rules_for_organization()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO rf_detection_rules (
        organization_id, name, description, enabled, metric, operator, threshold, severity,
        cooldown_seconds, auto_incident
    ) VALUES
      (NEW.id, 'High RF deviation', 'Correlates high-confidence RF deviations across supported metrics.', true, 'any', 'abs_z_gte', 6, 'high', 120, true),
      (NEW.id, 'Critical RF deviation', 'Escalates extreme RF deviations to critical incidents.', true, 'any', 'abs_z_gte', 8, 'critical', 120, true)
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS organizations_seed_detection_rules ON organizations;
CREATE TRIGGER organizations_seed_detection_rules
AFTER INSERT ON organizations
FOR EACH ROW EXECUTE FUNCTION lyrahcle_seed_detection_rules_for_organization();
