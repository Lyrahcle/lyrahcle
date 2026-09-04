-- External community reception evidence (SatNOGS Network and future equivalents).
-- These rows are third-party context: they show that a downlink was observable by
-- someone else, never that the spacecraft is healthy, compromised or vulnerable.

CREATE TABLE IF NOT EXISTS external_rf_observations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    satellite_id UUID NOT NULL REFERENCES satellites(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    external_id BIGINT NOT NULL,
    station_id BIGINT NOT NULL DEFAULT 0,
    station_name TEXT NOT NULL DEFAULT '',
    observer TEXT NOT NULL DEFAULT '',
    transmitter_uuid TEXT NOT NULL DEFAULT '',
    mode TEXT NOT NULL DEFAULT '',
    downlink_hz BIGINT,
    station_latitude DOUBLE PRECISION,
    station_longitude DOUBLE PRECISION,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT '',
    vetted_status TEXT NOT NULL DEFAULT '',
    reception_confirmed BOOLEAN NOT NULL DEFAULT false,
    evidence_kind TEXT NOT NULL DEFAULT 'external_community',
    applicability TEXT NOT NULL DEFAULT 'reception_context_only',
    confidence SMALLINT NOT NULL DEFAULT 30,
    waterfall_url TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE external_rf_observations
    DROP CONSTRAINT IF EXISTS external_rf_observations_source_check,
    ADD CONSTRAINT external_rf_observations_source_check
        CHECK (source IN ('satnogs_network')),
    DROP CONSTRAINT IF EXISTS external_rf_observations_evidence_kind_check,
    ADD CONSTRAINT external_rf_observations_evidence_kind_check
        CHECK (evidence_kind = 'external_community'),
    DROP CONSTRAINT IF EXISTS external_rf_observations_applicability_check,
    ADD CONSTRAINT external_rf_observations_applicability_check
        CHECK (applicability = 'reception_context_only'),
    DROP CONSTRAINT IF EXISTS external_rf_observations_confidence_check,
    ADD CONSTRAINT external_rf_observations_confidence_check
        CHECK (confidence BETWEEN 0 AND 60),
    DROP CONSTRAINT IF EXISTS external_rf_observations_window_check,
    ADD CONSTRAINT external_rf_observations_window_check
        CHECK (ended_at > started_at);

CREATE UNIQUE INDEX IF NOT EXISTS external_rf_observations_unique_source_record
    ON external_rf_observations (organization_id, source, external_id);

CREATE INDEX IF NOT EXISTS external_rf_observations_satellite_recent
    ON external_rf_observations (organization_id, satellite_id, ended_at DESC);

ALTER TABLE external_rf_observations ENABLE ROW LEVEL SECURITY;
ALTER TABLE external_rf_observations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS lyrahcle_tenant_isolation ON external_rf_observations;
CREATE POLICY lyrahcle_tenant_isolation ON external_rf_observations
    FOR ALL
    USING (lyrahcle_rls_allows_organization(organization_id))
    WITH CHECK (lyrahcle_rls_allows_organization(organization_id));

COMMENT ON TABLE external_rf_observations IS
    'Third-party ground reception records imported from community networks. Context only: never treated as measured mission evidence or as proof of onboard state.';
COMMENT ON COLUMN external_rf_observations.confidence IS
    'Bounded at 60 because the observation was neither made nor calibrated by a Lyrahcle station.';
COMMENT ON COLUMN external_rf_observations.reception_confirmed IS
    'True only when the upstream network vetted the observation as a good reception.';
