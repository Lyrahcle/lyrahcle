CREATE TABLE IF NOT EXISTS orbital_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    satellite_id UUID NOT NULL REFERENCES satellites(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('maneuver', 'conjunction')),
    event_time TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    severity TEXT NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orbital_events_satellite_time
    ON orbital_events (satellite_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_orbital_events_type_time
    ON orbital_events (event_type, event_time DESC);
