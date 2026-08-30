# Administrative audit coverage

Security-sensitive administrative mutations are audited in the organization they
modify.

Covered control-plane and RBAC actions:

- organization creation -> `organization.created`
- custom role creation -> `rbac.role_created`
- role assignment -> `rbac.role_assigned`
- role removal -> `rbac.role_removed`

Each RBAC event records the actor, target organization, affected role, and affected
user where applicable. Organization creation is written explicitly into the newly
created organization's audit stream and also records the actor's source organization
as `actor_organization_id`.

RBAC read operations (`/rbac/permissions` and `/rbac/roles`) use `rbac.read`; writes
continue to require `rbac.manage`.
