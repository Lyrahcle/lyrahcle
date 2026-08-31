# Sensitive endpoint audit

This inventory defines the minimum audit expectation for security-sensitive operations. Authorization enforcement remains server-side and audit records must not contain passwords, API keys, JWTs, WebAuthn challenge data, TOTP secrets or password-reset tokens.

| Area | Sensitive operation | Expected audit event |
| --- | --- | --- |
| Authentication | successful login | `auth.login` |
| Authentication | failed login | `auth.login_failed` |
| Authentication | logout | `auth.logout` |
| Authentication | password change | `auth.password_changed` |
| Authentication | TOTP enable/disable | `auth.totp_enabled` / `auth.totp_disabled` |
| Authentication | WebAuthn register/remove/login | `auth.webauthn_registered` / `auth.webauthn_removed` / `auth.webauthn_login` |
| Recovery | reset requested/completed | `auth.password_reset_requested` / `auth.password_reset_completed` |
| Sessions | revoke one/all | `auth.session_revoked` / `auth.sessions_revoked` |
| Users | create | `user.created` |
| Users | role change | `user.role_changed` |
| Users | enable/disable | `user.status_changed` |
| Users | delete | `user.deleted` |
| Agents | key create/revoke | `agent_key.created` / `agent_key.revoked` |
| Agents | failed key authentication | `agent_key.auth_failed` |
| Agents | fleet rollout create/start/advance/pause/resume/rollback | `agent.rollout_created` / `agent.rollout_started` / `agent.rollout_advanced` / `agent.rollout_paused` / `agent.rollout_resumed` / `agent.rollout_rolled_back` |
| Agents | enrollment token issue | `agent.enrollment_issued` |
| Agents | station update | `agent.station_updated` |
| Agents | status change | `agent.status_changed` |
| Agents | config issue/rollback | `agent.config_issued` / `agent.config_rollback` |
| Organizations | create/update | `organization.created` / `organization.updated` |
| Platform admins | grant/revoke | `platform_admin.granted` / `platform_admin.revoked` |
| Signing keys | policy update/prepare/activate/revoke | `signing_key.policy_updated` / `signing_key.prepared` / `signing_key.activated` / `signing_key.revoked` |
| Satellites | create/update TLE | `satellite.upserted` |
| Detection | create/update rule | `detection_rule.created` / `detection_rule.updated` |
| Incidents | create/update | `incident.created` / `incident.updated` |
| Incidents | note/evidence | `incident.note_added` / `incident.evidence_added` |
| Asset alerts | acknowledge | `asset_alert.acknowledged` |

Read-only endpoints are not audited individually unless access itself is security-significant. Audit-log access should be monitored at the platform/logging layer to avoid recursively writing audit events while listing the audit log.

Review this table whenever a mutating authentication, authorization, credential, agent, detection or incident endpoint is added.

## v9.0 SDR agent identity endpoints

| Endpoint | Authentication | Security property |
|---|---|---|
| `POST /api/v1/agents/enrollment-tokens` | Admin JWT | Issues one single-use token; only SHA-256 is persisted |
| `POST /api/v1/agents/enroll` | One-time enrollment token + TLS | CSR public key is rebound to server-selected `agent_id` |
| `GET /api/v1/agents/` | Admin JWT | Certificate and posture metadata only; no private keys or enrollment tokens |
| `PATCH /api/v1/agents/{agentID}/status` | Admin JWT | Quarantine denies authentication immediately; revocation is irreversible |
| `POST /api/v1/agents/me/certificate/rotate` | Valid active mTLS certificate | New key is generated on-agent; current certificate must authenticate rotation |
| `POST /api/v1/agents/me/device` | mTLS or legacy agent key | Identity comes from the credential, never from the request body |
| `POST /api/v1/rf/ingest` | mTLS, legacy agent key, or operator JWT | Agent-scoped credentials cannot submit another `agent_id` |
