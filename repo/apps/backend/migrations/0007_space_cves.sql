CREATE TABLE IF NOT EXISTS space_cves (
    cve_id      TEXT PRIMARY KEY,
    description TEXT NOT NULL,
    published   TIMESTAMPTZ,
    severity    TEXT NOT NULL DEFAULT 'unknown',
    score       DOUBLE PRECISION,
    keyword     TEXT NOT NULL,
    synced_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_space_cves_published ON space_cves(published DESC);
CREATE INDEX IF NOT EXISTS idx_space_cves_severity ON space_cves(severity);
