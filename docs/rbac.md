# Granular RBAC

## v11.0 — foundation
Organization-scoped permissions, custom roles, assignments and compatibility mappings.

## v11.1 — permission enforcement
Sensitive HTTP operations now enforce explicit capabilities. Reads use capabilities such as `assets.read`, `incidents.read`, `agents.read`, `audit.read`, `rf.read` and `orbital.read`. Mutations use `*.manage`, plus `rf.ingest`.

`/rbac/me` stays available to authenticated users for frontend capability discovery. RBAC administration requires `rbac.manage`.

Legacy JWT roles remain in tokens for compatibility, but endpoint authorization can now evolve independently of the old fixed role hierarchy.
