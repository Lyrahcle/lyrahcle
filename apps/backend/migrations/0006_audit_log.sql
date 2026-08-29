CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_type  TEXT NOT NULL,
    actor_id    TEXT NOT NULL,
    action      TEXT NOT NULL,
    ip          TEXT NOT NULL DEFAULT '',
    details     JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_audit_log_occurred_at ON audit_log(occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor ON audit_log(actor_type, actor_id);
