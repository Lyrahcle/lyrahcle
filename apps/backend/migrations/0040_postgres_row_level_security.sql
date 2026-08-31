-- v14.2: database-enforced tenant isolation for core operational data.
-- HTTP requests are marked by the backend pool hook. Requests without an
-- authenticated organization fail closed; non-HTTP system/migration contexts
-- retain explicit cross-tenant access for trusted background work.

CREATE OR REPLACE FUNCTION lyrahcle_rls_allows_organization(row_organization_id UUID)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT current_setting('lyrahcle.rls_request', true) IS DISTINCT FROM 'on'
        OR (
            NULLIF(current_setting('lyrahcle.organization_id', true), '') IS NOT NULL
            AND row_organization_id = NULLIF(current_setting('lyrahcle.organization_id', true), '')::uuid
        );
$$;

DO $$
DECLARE
    target_table text;
BEGIN
    FOREACH target_table IN ARRAY ARRAY[
        'programs',
        'assets',
        'incidents',
        'audit_log',
        'rf_readings',
        'rf_anomalies',
        'orbital_events',
        'rf_detection_rules',
        'rf_detection_matches',
        'organization_roles',
        'organization_membership_roles',
        'agent_fleet_rollouts',
        'agent_fleet_rollout_targets'
    ]
    LOOP
        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = target_table
              AND column_name = 'organization_id'
        ) THEN
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', target_table);
            EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', target_table);
            EXECUTE format('DROP POLICY IF EXISTS lyrahcle_tenant_isolation ON %I', target_table);
            EXECUTE format(
                'CREATE POLICY lyrahcle_tenant_isolation ON %I FOR ALL USING (lyrahcle_rls_allows_organization(organization_id)) WITH CHECK (lyrahcle_rls_allows_organization(organization_id))',
                target_table
            );
        END IF;
    END LOOP;
END $$;

-- Child tables that inherit tenant ownership from a protected parent.
ALTER TABLE incident_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE incident_events FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lyrahcle_tenant_isolation ON incident_events;
CREATE POLICY lyrahcle_tenant_isolation ON incident_events
FOR ALL
USING (
    current_setting('lyrahcle.rls_request', true) IS DISTINCT FROM 'on'
    OR EXISTS (
        SELECT 1 FROM incidents i
        WHERE i.id = incident_events.incident_id
          AND lyrahcle_rls_allows_organization(i.organization_id)
    )
)
WITH CHECK (
    current_setting('lyrahcle.rls_request', true) IS DISTINCT FROM 'on'
    OR EXISTS (
        SELECT 1 FROM incidents i
        WHERE i.id = incident_events.incident_id
          AND lyrahcle_rls_allows_organization(i.organization_id)
    )
);

ALTER TABLE asset_cve_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_cve_alerts FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lyrahcle_tenant_isolation ON asset_cve_alerts;
CREATE POLICY lyrahcle_tenant_isolation ON asset_cve_alerts
FOR ALL
USING (
    current_setting('lyrahcle.rls_request', true) IS DISTINCT FROM 'on'
    OR EXISTS (
        SELECT 1 FROM assets a
        WHERE a.id = asset_cve_alerts.asset_id
          AND lyrahcle_rls_allows_organization(a.organization_id)
    )
)
WITH CHECK (
    current_setting('lyrahcle.rls_request', true) IS DISTINCT FROM 'on'
    OR EXISTS (
        SELECT 1 FROM assets a
        WHERE a.id = asset_cve_alerts.asset_id
          AND lyrahcle_rls_allows_organization(a.organization_id)
    )
);
