-- v11.8: separate global control-plane administration from tenant-local RBAC.

CREATE TABLE IF NOT EXISTS platform_administrators (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Upgrade path: preserve control-plane access by promoting only the oldest active
-- admin in the default organization when no platform administrator exists yet.
INSERT INTO platform_administrators (user_id)
SELECT u.id
FROM users u
WHERE u.organization_id = '00000000-0000-0000-0000-000000000001'
  AND u.role = 'admin'
  AND u.enabled = true
  AND u.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM platform_administrators)
ORDER BY u.created_at ASC
LIMIT 1
ON CONFLICT DO NOTHING;
