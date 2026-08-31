ALTER TABLE agent_config_versions
    ADD COLUMN IF NOT EXISTS signing_kid TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'agent_config_versions_signing_kid_format'
    ) THEN
        ALTER TABLE agent_config_versions
            ADD CONSTRAINT agent_config_versions_signing_kid_format
            CHECK (
                signing_kid = ''
                OR signing_kid ~ '^[A-Za-z0-9._:-]{1,128}$'
            );
    END IF;
END $$;

-- Existing rows keep the empty-string legacy marker, but future inserts must
-- provide signing_kid explicitly so new signed versions cannot silently omit it.
ALTER TABLE agent_config_versions
    ALTER COLUMN signing_kid DROP DEFAULT;

COMMENT ON COLUMN agent_config_versions.signing_kid IS
    'Stable key identifier for the signing key. Empty string marks pre-KID legacy versions.';
