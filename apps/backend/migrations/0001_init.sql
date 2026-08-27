-- Lyrahcle core schema (Phase 3)

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS programs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id  UUID REFERENCES programs(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,       -- domain | ip | satellite | ground_station | firmware
    value       TEXT NOT NULL,       -- e.g. hostname, IP, NORAD ID
    first_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (program_id, kind, value)
);

CREATE TABLE IF NOT EXISTS scans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id  UUID REFERENCES programs(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending', -- pending | running | done | failed
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    checkpoint  JSONB NOT NULL DEFAULT '{}'::jsonb, -- for resuming after restart
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS findings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id     UUID REFERENCES scans(id) ON DELETE CASCADE,
    asset_id    UUID REFERENCES assets(id) ON DELETE CASCADE,
    severity    TEXT NOT NULL DEFAULT 'info', -- info | low | medium | high | critical
    title       TEXT NOT NULL,
    details     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assets_program ON assets(program_id);
CREATE INDEX IF NOT EXISTS idx_scans_program ON scans(program_id);
CREATE INDEX IF NOT EXISTS idx_findings_scan ON findings(scan_id);
