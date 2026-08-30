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
