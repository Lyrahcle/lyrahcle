CREATE TABLE IF NOT EXISTS satellites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    norad_id    INTEGER NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    tle_line1   TEXT NOT NULL,
    tle_line2   TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
