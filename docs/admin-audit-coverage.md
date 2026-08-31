# Administrative audit coverage

Security-sensitive administrative mutations are audited in the organization they
modify.

Covered control-plane and RBAC actions:

- organization creation -> `organization.created`
- custom role creation -> `rbac.role_created`
- role assignment -> `rbac.role_assigned`
- role removal -> `rbac.role_removed`
- fleet rollout creation -> `agent.rollout_created`
- fleet rollout start -> `agent.rollout_started`
- fleet rollout advance -> `agent.rollout_advanced`
- fleet rollout pause -> `agent.rollout_paused`
- fleet rollout resume -> `agent.rollout_resumed`
- fleet rollout rollback -> `agent.rollout_rolled_back`

Each RBAC event records the actor, target organization, affected role, and affected
user where applicable. Organization creation is written explicitly into the newly
created organization's audit stream and also records the actor's source organization
as `actor_organization_id`.

RBAC read operations (`/rbac/permissions` and `/rbac/roles`) use `rbac.read`; writes
continue to require `rbac.manage`. Fleet rollout write operations require
`agents.manage`, record the authenticated actor in the tenant audit stream, and use
centralized audit action constants so transition names cannot silently drift.

`fleet_rollout_audit_v145_test.go` is the machine-verifiable contract for the full
rollout transition set. Adding or removing a rollout state-changing endpoint requires
updating that contract and this inventory.
