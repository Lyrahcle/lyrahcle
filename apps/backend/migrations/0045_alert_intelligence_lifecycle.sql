ALTER TABLE assets
  ADD COLUMN IF NOT EXISTS cpe_uri TEXT NOT NULL DEFAULT '';

ALTER TABLE space_cves
  ADD COLUMN IF NOT EXISTS cpe_matches JSONB NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS kev BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS kev_due_date DATE,
  ADD COLUMN IF NOT EXISTS epss_score DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS epss_percentile DOUBLE PRECISION;

ALTER TABLE asset_cve_alerts
  ADD COLUMN IF NOT EXISTS lifecycle_status TEXT NOT NULL DEFAULT 'open',
  ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now();

DO $$ BEGIN
  ALTER TABLE asset_cve_alerts ADD CONSTRAINT asset_cve_alert_lifecycle_status_check
    CHECK (lifecycle_status IN ('open','reviewed','resolved','suppressed'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_assets_cpe_uri ON assets(cpe_uri) WHERE cpe_uri <> '';
CREATE INDEX IF NOT EXISTS idx_space_cves_kev ON space_cves(kev) WHERE kev = true;
CREATE INDEX IF NOT EXISTS idx_space_cves_epss ON space_cves(epss_score DESC) WHERE epss_score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_asset_cve_alerts_lifecycle ON asset_cve_alerts(lifecycle_status, created_at DESC);
