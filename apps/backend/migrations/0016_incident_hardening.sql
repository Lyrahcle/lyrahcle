ALTER TABLE incidents
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

ALTER TABLE incidents
    ADD CONSTRAINT incidents_idempotency_key_length_check
    CHECK (idempotency_key IS NULL OR char_length(idempotency_key) BETWEEN 1 AND 128);

CREATE UNIQUE INDEX IF NOT EXISTS idx_incidents_idempotency_key
    ON incidents(idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS incident_evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    evidence_type TEXT NOT NULL,
    evidence_ref TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT incident_evidence_type_check CHECK (evidence_type IN ('rf_anomaly', 'request', 'cve', 'agent', 'satellite', 'external')),
    CONSTRAINT incident_evidence_ref_length_check CHECK (char_length(evidence_ref) BETWEEN 1 AND 256),
    CONSTRAINT incident_evidence_label_length_check CHECK (char_length(label) <= 160),
    UNIQUE (incident_id, evidence_type, evidence_ref)
);

CREATE INDEX IF NOT EXISTS idx_incident_evidence_incident
    ON incident_evidence(incident_id, created_at ASC);

ALTER TABLE incident_events
    DROP CONSTRAINT IF EXISTS incident_events_type_check;

ALTER TABLE incident_events
    ADD CONSTRAINT incident_events_type_check
    CHECK (event_type IN ('created', 'status_changed', 'severity_changed', 'assignment_changed', 'note', 'auto_detected', 'evidence_added'));

CREATE OR REPLACE FUNCTION enforce_incident_status_transition()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = OLD.status THEN
        RETURN NEW;
    END IF;

    IF OLD.status = 'new' AND NEW.status IN ('investigating', 'false_positive') THEN
        RETURN NEW;
    END IF;

    IF OLD.status = 'investigating' AND NEW.status IN ('contained', 'resolved', 'false_positive') THEN
        RETURN NEW;
    END IF;

    IF OLD.status = 'contained' AND NEW.status IN ('investigating', 'resolved') THEN
        RETURN NEW;
    END IF;

    RAISE EXCEPTION 'invalid incident status transition: % -> %', OLD.status, NEW.status USING ERRCODE = '23514';
END;
$$;

DROP TRIGGER IF EXISTS trg_incidents_status_transition ON incidents;

CREATE TRIGGER trg_incidents_status_transition
BEFORE UPDATE OF status ON incidents
FOR EACH ROW
EXECUTE FUNCTION enforce_incident_status_transition();
