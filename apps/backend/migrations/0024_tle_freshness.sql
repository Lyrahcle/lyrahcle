ALTER TABLE satellites ADD COLUMN IF NOT EXISTS tle_epoch TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_satellites_tle_epoch ON satellites (tle_epoch);
