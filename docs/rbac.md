# Granular RBAC

v11.0 introduces organization-scoped capabilities while preserving the existing coarse roles for compatibility.

## Model

- `permissions`: stable capability keys such as `assets.read` or `agents.manage`.
- `organization_roles`: system or custom roles inside one organization.
- `organization_role_permissions`: capability assignments to a role.
- `organization_membership_roles`: role assignments for a user in an organization.

Existing `viewer`, `auditor`, `analyst`, `operator` and `admin` roles are seeded as system roles for every organization and users are backfilled into the matching compatibility role.

## Authorization

The new `auth.RequirePermission` middleware can enforce capabilities directly from the authenticated user and active `organization_id`.

v11.0 keeps legacy role middleware in place to avoid a flag-day migration. New and migrated sensitive endpoints should move to capability checks incrementally.

## Admin API

Admin-only RBAC routes expose:

- list permissions;
- list organization roles;
- create custom organization roles;
- inspect current effective permissions;
- assign/remove organization roles from users.

The next phase should migrate endpoint authorization from role thresholds to explicit permissions and add UI support for custom roles.
