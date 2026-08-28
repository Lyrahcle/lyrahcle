ALTER TABLE space_cves 
ADD COLUMN IF NOT EXISTS references TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN IF NOT EXISTS search_keyword VARCHAR(255) DEFAULT '',
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_space_cves_severity ON space_cves(severity);
CREATE INDEX IF NOT EXISTS idx_space_cves_synced_at ON space_cves(synced_at);
CREATE INDEX IF NOT EXISTS idx_space_cves_search_keyword ON space_cves(search_keyword);

CREATE OR REPLACE FUNCTION update_space_cves_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS space_cves_update_timestamp ON space_cves;
CREATE TRIGGER space_cves_update_timestamp
BEFORE UPDATE ON space_cves
FOR EACH ROW
EXECUTE FUNCTION update_space_cves_timestamp();
