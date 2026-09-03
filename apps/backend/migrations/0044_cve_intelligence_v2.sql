ALTER TABLE space_cves
  ADD COLUMN IF NOT EXISTS last_modified TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS vulnerability_status TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS source_identifier TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS cvss_version TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS cvss_vector TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS cwe_ids TEXT[] NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS cve_sync_state (
  keyword TEXT PRIMARY KEY,
  last_modified_cursor TIMESTAMPTZ,
  last_success_at TIMESTAMPTZ,
  pages_fetched INTEGER NOT NULL DEFAULT 0 CHECK (pages_fetched >= 0),
  records_fetched INTEGER NOT NULL DEFAULT 0 CHECK (records_fetched >= 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE cve_sync_state IS 'Global NVD ingestion cursors. Not tenant data and only accessed by the CVE worker.';
