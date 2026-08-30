# Platform administrator boundary

Global organization management is a control-plane capability and is no longer
authorized through tenant-local RBAC permissions.

`platform_administrators` is the explicit control-plane allowlist. Tenant admins
continue to manage users, agents, incidents, assets, RBAC and other resources
inside their own organization, but cannot list or create organizations merely
because they hold the tenant `admin` compatibility role.

Upgrade behavior promotes only the oldest active admin in the default organization
when no platform administrator exists. Fresh installations grant platform
administrator status to the initial seeded administrator.

Future control-plane APIs should reuse this boundary rather than
`organizations.manage`.

## Organization metadata management

Platform administrators can list, create and update organization metadata from the Platform UI. The update surface changes only `slug` and `name`; the existing `enabled` database field is displayed as lifecycle state but is intentionally read-only until disable/enable semantics are enforced consistently for human sessions, machine credentials, background workers and recovery access.

The global routes remain behind the explicit platform-administrator boundary. Tenant-local `organizations.read` / `organizations.manage` capabilities must not grant access to `/api/v1/organizations/*` control-plane mutations.
