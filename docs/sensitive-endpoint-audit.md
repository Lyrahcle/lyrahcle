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
| Satellites | create/update TLE | `satellite.upserted` |
| Detection | create/update rule | `detection_rule.created` / `detection_rule.updated` |
| Incidents | create/update | `incident.created` / `incident.updated` |
| Incidents | note/evidence | `incident.note_added` / `incident.evidence_added` |
| Asset alerts | acknowledge | `asset_alert.acknowledged` |

Read-only endpoints are not audited individually unless access itself is security-significant. Audit-log access should be monitored at the platform/logging layer to avoid recursively writing audit events while listing the audit log.

Review this table whenever a mutating authentication, authorization, credential, agent, detection or incident endpoint is added.
