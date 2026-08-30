# Organizations and multi-tenancy

## v10.0 foundation

v10.0 introduced organizations, default-organization migration, user membership and JWT organization context.

## v10.1 user administration isolation

The administrative user surface is now tenant-scoped:

- user listing is restricted to the authenticated organization;
- newly created users are assigned to the administrator's organization;
- role changes, enable/disable and soft-delete require the target user to belong to the same organization;
- the last-admin safeguard is calculated per organization;
- user-administration audit metadata records `organization_id`;
- organization-less legacy tokens are rejected by tenant administrative routes.

A known user UUID from another organization is intentionally treated as not found by scoped store methods.

## Remaining isolation work

Complete multi-tenancy still requires organization scoping for agents, assets, incidents, audit queries, RF, orbital data, programs, CVE relationships, background jobs, search and streaming/SSE. Those domains must receive negative cross-tenant tests before production multi-tenancy is declared complete.
