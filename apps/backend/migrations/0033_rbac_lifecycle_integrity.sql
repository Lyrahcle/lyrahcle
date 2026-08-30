-- v11.3.3: keep compatibility roles and granular RBAC lifecycle in sync.

CREATE OR REPLACE FUNCTION lyrahcle_seed_organization_rbac(target_organization_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO organization_roles (organization_id, name, description, system)
    VALUES
      (target_organization_id, 'viewer', 'Compatibility role for viewer', true),
      (target_organization_id, 'auditor', 'Compatibility role for auditor', true),
      (target_organization_id, 'analyst', 'Compatibility role for analyst', true),
      (target_organization_id, 'operator', 'Compatibility role for operator', true),
      (target_organization_id, 'admin', 'Compatibility role for admin', true)
    ON CONFLICT (organization_id, name) DO NOTHING;

    INSERT INTO organization_role_permissions (organization_role_id, permission_key)
    SELECT r.id, p.key
    FROM organization_roles r
    JOIN permissions p ON (
        r.name = 'admin'
        OR (r.name = 'viewer' AND p.key = ANY(ARRAY[
            'assets.read','incidents.read','programs.read','rf.read','orbital.read','agents.read'
        ]))
        OR (r.name = 'auditor' AND p.key = ANY(ARRAY[
            'assets.read','incidents.read','programs.read','rf.read','orbital.read','agents.read','audit.read'
        ]))
        OR (r.name = 'analyst' AND p.key = ANY(ARRAY[
            'assets.read','incidents.read','incidents.manage','programs.read','rf.read','orbital.read','agents.read','audit.read'
        ]))
        OR (r.name = 'operator' AND p.key = ANY(ARRAY[
            'assets.read','assets.manage','incidents.read','incidents.manage','programs.read','programs.manage',
            'rf.read','rf.ingest','orbital.read','orbital.manage','agents.read','agents.manage','audit.read'
        ]))
    )
    WHERE r.organization_id = target_organization_id
      AND r.system = true
      AND r.name IN ('viewer','auditor','analyst','operator','admin')
    ON CONFLICT DO NOTHING;
END;
$$;

-- Repair/seed every organization that already exists.
SELECT lyrahcle_seed_organization_rbac(id) FROM organizations;

CREATE OR REPLACE FUNCTION lyrahcle_seed_new_organization_rbac()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM lyrahcle_seed_organization_rbac(NEW.id);
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS organizations_seed_rbac ON organizations;
CREATE TRIGGER organizations_seed_rbac
AFTER INSERT ON organizations
FOR EACH ROW EXECUTE FUNCTION lyrahcle_seed_new_organization_rbac();

CREATE OR REPLACE FUNCTION lyrahcle_sync_user_compatibility_role()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    -- Only manage the five compatibility system roles. Custom roles are preserved.
    DELETE FROM organization_membership_roles umr
    USING organization_roles r
    WHERE umr.user_id = NEW.id
      AND umr.organization_role_id = r.id
      AND r.system = true
      AND r.name IN ('viewer','auditor','analyst','operator','admin');

    IF NEW.deleted_at IS NULL THEN
        INSERT INTO organization_membership_roles (user_id, organization_id, organization_role_id)
        SELECT NEW.id, NEW.organization_id, r.id
        FROM organization_roles r
        WHERE r.organization_id = NEW.organization_id
          AND r.name = NEW.role
          AND r.system = true
        ON CONFLICT DO NOTHING;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS users_sync_compatibility_role ON users;
CREATE TRIGGER users_sync_compatibility_role
AFTER INSERT OR UPDATE OF role, organization_id, deleted_at ON users
FOR EACH ROW EXECUTE FUNCTION lyrahcle_sync_user_compatibility_role();

-- Repair users created after 0032 on databases where RBAC was already present.
INSERT INTO organization_membership_roles (user_id, organization_id, organization_role_id)
SELECT u.id, u.organization_id, r.id
FROM users u
JOIN organization_roles r
  ON r.organization_id = u.organization_id
 AND r.name = u.role
 AND r.system = true
WHERE u.deleted_at IS NULL
ON CONFLICT DO NOTHING;

-- Enforce that membership user/role rows belong to the same organization.
ALTER TABLE users
    ADD CONSTRAINT users_id_organization_id_key UNIQUE (id, organization_id);
ALTER TABLE organization_roles
    ADD CONSTRAINT organization_roles_id_organization_id_key UNIQUE (id, organization_id);
ALTER TABLE organization_membership_roles
    ADD CONSTRAINT organization_membership_roles_user_org_fkey
    FOREIGN KEY (user_id, organization_id)
    REFERENCES users(id, organization_id)
    ON DELETE CASCADE;
ALTER TABLE organization_membership_roles
    ADD CONSTRAINT organization_membership_roles_role_org_fkey
    FOREIGN KEY (organization_role_id, organization_id)
    REFERENCES organization_roles(id, organization_id)
    ON DELETE CASCADE;
