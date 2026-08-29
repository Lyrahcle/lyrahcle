ALTER TABLE users ADD COLUMN IF NOT EXISTS username TEXT;

-- Backfill existing rows with a username derived from the email local part.
-- Collisions (two emails sharing a local part) are disambiguated with a
-- short suffix taken from the user id.
WITH derived AS (
    SELECT
        id,
        regexp_replace(lower(split_part(email, '@', 1)), '[^a-z0-9_.-]', '', 'g') AS base
    FROM users
    WHERE username IS NULL
)
UPDATE users u
SET username = CASE
    WHEN (SELECT count(*) FROM derived d2 WHERE d2.base = d.base) > 1
        THEN d.base || '-' || substr(u.id::text, 1, 8)
    ELSE d.base
END
FROM derived d
WHERE u.id = d.id;

ALTER TABLE users ALTER COLUMN username SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_username_key ON users (username);
