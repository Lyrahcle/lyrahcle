ALTER TABLE space_cves
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'nvd';

ALTER TABLE asset_cve_alerts
    ADD COLUMN IF NOT EXISTS applicability_status TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS match_method TEXT NOT NULL DEFAULT 'thematic_asset_kind',
    ADD COLUMN IF NOT EXISTS confidence SMALLINT NOT NULL DEFAULT 20,
    ADD COLUMN IF NOT EXISTS context_only BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE asset_cve_alerts
    DROP CONSTRAINT IF EXISTS asset_cve_alerts_applicability_status_check,
    ADD CONSTRAINT asset_cve_alerts_applicability_status_check
        CHECK (applicability_status IN ('unknown', 'potentially_relevant', 'applicable', 'probable_exposure', 'confirmed', 'not_affected')),
    DROP CONSTRAINT IF EXISTS asset_cve_alerts_confidence_check,
    ADD CONSTRAINT asset_cve_alerts_confidence_check
        CHECK (confidence BETWEEN 0 AND 100);

COMMENT ON COLUMN asset_cve_alerts.context_only IS
    'True when the association is contextual and must not be presented as a confirmed vulnerability.';
COMMENT ON COLUMN asset_cve_alerts.match_method IS
    'Explainable method used to associate the CVE with the asset.';

