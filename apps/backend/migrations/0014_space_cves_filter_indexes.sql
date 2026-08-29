CREATE INDEX IF NOT EXISTS idx_space_cves_keyword ON space_cves(keyword);
CREATE INDEX IF NOT EXISTS idx_space_cves_score ON space_cves(score DESC NULLS LAST);
