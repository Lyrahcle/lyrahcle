-- v11.0: granular RBAC foundation.
-- Existing coarse roles remain supported through seeded compatibility mappings.

CREATE TABLE IF NOT EXISTS permissions (
    key TEXT PRIMARY KEY,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS organization_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);

CREATE TABLE IF NOT EXISTS organization_role_permissions (
    organization_role_id UUID NOT NULL REFERENCES organization_roles(id) ON DELETE CASCADE,
    permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
    PRIMARY KEY (organization_role_id, permission_key)
);

CREATE TABLE IF NOT EXISTS organization_membership_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    organization_role_id UUID NOT NULL REFERENCES organization_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, organization_id, organization_role_id)
);

INSERT INTO permissions (key, description) VALUES
('users.read','Read organization users'),
('users.manage','Create/update/deactivate organization users'),
('agents.read','Read organization agents'),
('agents.manage','Manage enrollment, station, status and configuration'),
('assets.read','Read organization assets'),
('assets.manage','Create/update organization assets'),
('incidents.read','Read organization incidents'),
('incidents.manage','Create/update organization incidents'),
('audit.read','Read organization audit history'),
('programs.read','Read organization programs'),
('programs.manage','Create/update organization programs'),
('rf.read','Read organization RF telemetry/anomalies'),
('rf.ingest','Ingest RF telemetry'),
('orbital.read','Read orbital context'),
('orbital.manage','Create trusted/operator orbital context'),
('organizations.read','Read organization metadata'),
('organizations.manage','Manage organizations'),
('rbac.read','Read RBAC roles and permissions'),
('rbac.manage','Manage RBAC roles and permission assignments')
ON CONFLICT (key) DO NOTHING;

-- Seed compatibility system roles for every existing organization.
INSERT INTO organization_roles (organization_id,name,description,system)
SELECT o.id, r.name, r.description, true
FROM organizations o
CROSS JOIN (VALUES
 ('viewer','Compatibility role for viewer'),
 ('auditor','Compatibility role for auditor'),
 ('analyst','Compatibility role for analyst'),
 ('operator','Compatibility role for operator'),
 ('admin','Compatibility role for admin')
) AS r(name,description)
ON CONFLICT (organization_id,name) DO NOTHING;

-- Viewer baseline.
INSERT INTO organization_role_permissions (organization_role_id,permission_key)
SELECT r.id,p.key
FROM organization_roles r
JOIN permissions p ON p.key = ANY(ARRAY[
 'assets.read','incidents.read','programs.read','rf.read','orbital.read','agents.read'
])
WHERE r.name='viewer'
ON CONFLICT DO NOTHING;

-- Auditor = viewer + audit.
INSERT INTO organization_role_permissions (organization_role_id,permission_key)
SELECT r.id,p.key
FROM organization_roles r
JOIN permissions p ON p.key = ANY(ARRAY[
 'assets.read','incidents.read','programs.read','rf.read','orbital.read','agents.read','audit.read'
])
WHERE r.name='auditor'
ON CONFLICT DO NOTHING;

-- Analyst = viewer + incident management + audit.
INSERT INTO organization_role_permissions (organization_role_id,permission_key)
SELECT r.id,p.key
FROM organization_roles r
JOIN permissions p ON p.key = ANY(ARRAY[
 'assets.read','incidents.read','incidents.manage','programs.read','rf.read','orbital.read','agents.read','audit.read'
])
WHERE r.name='analyst'
ON CONFLICT DO NOTHING;

-- Operator = analyst + operational management.
INSERT INTO organization_role_permissions (organization_role_id,permission_key)
SELECT r.id,p.key
FROM organization_roles r
JOIN permissions p ON p.key = ANY(ARRAY[
 'assets.read','assets.manage','incidents.read','incidents.manage','programs.read','programs.manage',
 'rf.read','rf.ingest','orbital.read','orbital.manage','agents.read','agents.manage','audit.read'
])
WHERE r.name='operator'
ON CONFLICT DO NOTHING;

-- Admin = all current permissions.
INSERT INTO organization_role_permissions (organization_role_id,permission_key)
SELECT r.id,p.key
FROM organization_roles r
CROSS JOIN permissions p
WHERE r.name='admin'
ON CONFLICT DO NOTHING;

-- Backfill each existing user into the compatibility role matching users.role.
INSERT INTO organization_membership_roles (user_id,organization_id,organization_role_id)
SELECT u.id,u.organization_id,r.id
FROM users u
JOIN organization_roles r
  ON r.organization_id=u.organization_id AND r.name=u.role
WHERE u.deleted_at IS NULL
ON CONFLICT DO NOTHING;

CREATE INDEX IF NOT EXISTS org_membership_roles_user_org_idx
    ON organization_membership_roles (user_id, organization_id);
CREATE INDEX IF NOT EXISTS org_role_permissions_role_idx
    ON organization_role_permissions (organization_role_id);
