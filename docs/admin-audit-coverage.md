# Administrative audit coverage

Security-sensitive administrative mutations are audited in the organization they
modify.

Covered control-plane, identity and RBAC actions:

- organization creation -> `organization.created`
- organization update -> `organization.updated`
- platform-admin grant -> `platform_admin.granted`
- platform-admin revoke -> `platform_admin.revoked`
- user creation -> `user.created`
- user role change -> `user.role_changed`
- user enable/disable -> `user.status_changed`
- user soft-delete -> `user.deleted`
- custom role creation -> `rbac.role_created`
- role assignment -> `rbac.role_assigned`
- role removal -> `rbac.role_removed`
- fleet rollout creation -> `agent.rollout_created`
- fleet rollout start -> `agent.rollout_started`
- fleet rollout advance -> `agent.rollout_advanced`
- fleet rollout pause -> `agent.rollout_paused`
- fleet rollout resume -> `agent.rollout_resumed`
- fleet rollout rollback -> `agent.rollout_rolled_back`
- fleet rollout fail-closed transition -> `agent.rollout_failed`
- agent enrollment token issue -> `agent.enrollment_issued`
- agent station update -> `agent.station_updated`
- agent status change -> `agent.status_changed`
- agent config issue -> `agent.config_issued`
- agent config rollback -> `agent.config_rollback`
- signing-key policy update -> `signing_key.policy_updated`
- signing-key prepare -> `signing_key.prepared`
- signing-key activate -> `signing_key.activated`
- signing-key revoke -> `signing_key.revoked`

Each administrative audit event records actor identity and source IP in the audit row.
The shared audit helpers also enrich event details with `organization_id`,
`result=success`, and the HTTP `request_id` when one is present, so privileged changes
can be correlated with request logs without relying on every route to add those fields
manually. RBAC events additionally record affected role and user where applicable.
Organization creation is written explicitly into the newly created organization's audit
stream and also records the actor's source organization as `actor_organization_id`.

RBAC read operations (`/rbac/permissions` and `/rbac/roles`) use `rbac.read`; writes
continue to require `rbac.manage`. Fleet rollout write operations require
`agents.manage`, record the authenticated actor in the tenant audit stream, and use
centralized audit action constants so transition names cannot silently drift. Automatic fail-closed transitions caused by rejection thresholds, acknowledgement deadlines, target-policy drift, or configuration issuance failures are recorded as `agent.rollout_failed` with a machine-readable reason.

`fleet_rollout_audit_v145_test.go` is the machine-verifiable contract for the full
rollout transition set. `admin_audit_matrix_v1452_test.go` extends that contract across
organization/platform control-plane changes, signing-key lifecycle mutations, agent
identity administration, and remote-config issue/rollback operations. Adding or
removing a privileged state-changing endpoint requires updating the applicable contract
and this inventory.
