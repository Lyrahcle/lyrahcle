CREATE TABLE IF NOT EXISTS asset_cve_alerts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id         UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    cve_id           TEXT NOT NULL REFERENCES space_cves(cve_id) ON DELETE CASCADE,
    matched_keyword  TEXT NOT NULL,
    severity         TEXT NOT NULL,
    acknowledged     BOOLEAN NOT NULL DEFAULT false,
    acknowledged_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    acknowledged_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (asset_id, cve_id)
);

CREATE INDEX IF NOT EXISTS idx_asset_cve_alerts_asset ON asset_cve_alerts(asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_cve_alerts_created ON asset_cve_alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_asset_cve_alerts_unacked ON asset_cve_alerts(acknowledged) WHERE acknowledged = false;
