-- Scope HTTP idempotency keys to the tenant that issued them. A caller in one
-- organization must never collide with or receive an incident from another
-- organization merely by reusing the same Idempotency-Key value.
DROP INDEX IF EXISTS idx_incidents_idempotency_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_incidents_org_idempotency_key
    ON incidents(organization_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Background work may be delivered more than once. A nullable dedupe key lets
-- system jobs repair a partial side effect (for example, alert row committed
-- but audit write interrupted) without creating duplicate audit entries.
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS dedupe_key TEXT;

DO $$ BEGIN
  ALTER TABLE audit_log ADD CONSTRAINT audit_log_dedupe_key_length_check
    CHECK (dedupe_key IS NULL OR char_length(dedupe_key) BETWEEN 1 AND 256);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_log_org_dedupe_key
    ON audit_log(organization_id, dedupe_key)
    WHERE dedupe_key IS NOT NULL;
