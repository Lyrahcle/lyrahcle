CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'new',
    source_type TEXT NOT NULL DEFAULT 'manual',
    satellite_id UUID REFERENCES satellites(id) ON DELETE SET NULL,
    agent_id TEXT,
    source_anomaly_id UUID UNIQUE REFERENCES rf_anomalies(id) ON DELETE SET NULL,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    CONSTRAINT incidents_severity_check CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    CONSTRAINT incidents_status_check CHECK (status IN ('new', 'investigating', 'contained', 'resolved', 'false_positive')),
    CONSTRAINT incidents_source_type_check CHECK (source_type IN ('manual', 'rf_anomaly')),
    CONSTRAINT incidents_title_length_check CHECK (char_length(title) BETWEEN 1 AND 160),
    CONSTRAINT incidents_description_length_check CHECK (char_length(description) <= 4000),
    CONSTRAINT incidents_agent_id_length_check CHECK (agent_id IS NULL OR char_length(agent_id) BETWEEN 1 AND 128)
);

CREATE TABLE IF NOT EXISTS incident_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    message TEXT NOT NULL DEFAULT '',
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT incident_events_type_check CHECK (event_type IN ('created', 'status_changed', 'severity_changed', 'assignment_changed', 'note', 'auto_detected')),
    CONSTRAINT incident_events_message_length_check CHECK (char_length(message) <= 2000)
);

CREATE INDEX IF NOT EXISTS idx_incidents_created_at ON incidents(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_agent ON incidents(agent_id, created_at DESC) WHERE agent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_incidents_satellite ON incidents(satellite_id, created_at DESC) WHERE satellite_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_incidents_assigned_to ON incidents(assigned_to, created_at DESC) WHERE assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_incident_events_incident ON incident_events(incident_id, created_at ASC);
