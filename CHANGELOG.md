- v11.16: fixed PostgreSQL asset projection/scan parity exposed by CI integration tests and completed capability-based auth for hybrid RF ingest, detection rules, RF reads, alerts, and machine-only agent self-service routes.
- v11.1: enforce granular capabilities across sensitive HTTP read/write surfaces.
- v11.0.1: fixed backend import paths to match the Go module declared in apps/backend/go.mod.
- v11.0: granular organization-scoped RBAC foundation with permissions, custom roles, compatibility mappings and permission middleware.
- v10.6: added tenant-isolation audit gates, full tenant snapshots, source-contract checks and optional database-backed Org-A/Org-B integration tests.
- v10.5: tenant-isolated programs, RF telemetry/anomalies, orbital events, asset-CVE alert operations and mixed global/local search.
- v10.4: tenant-isolated agent administration and organization ownership across enrollment, certificates, remote config and legacy agent keys.
- v10.3: tenant-isolated assets, incidents and audit handlers; corrected v10.2 agent ownership migration to `agent_identities`.
- v10.2: added mandatory organization ownership and tenant-first indexes for agents, assets, incidents and audit, plus reusable tenant-scope query primitives.
- v10.1: tenant-scoped user administration with per-organization list/create/role/status/delete controls and last-admin protection.
- v10.0: Organizations / multi-tenancy foundation with default-org migration, user membership, JWT organization context and admin organization API.
# Changelog


## Unreleased

- Added v9.5 maneuver/conjunction context persistence and bounded viewer/operator API endpoints; events remain candidate-only evidence.
- Extended cross-domain correlation with satellite-specific orbital-operations evidence while preserving severity safety gates.
- Fixed v9.4 automatic evidence persistence to use the database-supported `external` evidence type instead of silently rejected custom types.

### v9.4 cross-domain correlation

- Added an explainable cross-domain correlation summary to automatically generated RF detection incidents.
- Correlation now reports confidence and corroborating domains across RF/detection rules, orbital evidence, agent posture, and recent agent audit activity.
- Recent unacknowledged asset CVE alerts can be attached as explicitly `context_only` evidence and never increase severity by themselves.
- Cross-domain severity remains candidate-only; existing orbital/TLE safety gates and incident severity policy remain authoritative.


- Added v9.3 satellite pass prediction with AOS/LOS, peak elevation, configurable station masks, bounded prediction horizons, and v9.2.2 TLE-quality gating.
### v9.2.2 TLE freshness and correlation safety

- Added TLE epoch persistence and configurable freshness states derived from the orbital element itself rather than ingestion time.
- Suppressed orbital severity escalation for aging, stale, unusable, or unvalidated orbital context and disabled orbital auto-escalation by default.
- Added TLE age/staleness and CelesTrak synchronization metrics plus Prometheus alerts for degraded orbital data quality.
- Exposed TLE quality and orbital-confidence fields in satellite responses and omitted propagated positions for stale/unusable TLEs.

- Refined the README into a concise project landing page with product context, CI/license/release/coverage badges, the existing dashboard screenshot, a development-only quick-start warning, visible contribution guidance, and direct roadmap links.

Notable changes are tracked here until versioned release notes take over.

## Previous unreleased milestones

### v9.1.1 backend runtime image hardening

- Replaced the backend distroless runtime dependency with a scratch image to remove the failing GCR runtime pull from CI.
- Copied only CA certificates, timezone data, the static Go binary, and migrations into the runtime image.
- Kept the backend non-root with numeric UID/GID 65532 and stripped build paths/symbols from the production binary.

### v9.1 secure remote agent management

- Added declarative, versioned SDR agent configuration for capture frequency and interval with strict server/database/agent bounds.
- Added independent Ed25519 signing of remote configuration; agents verify signature, checksum, and bound `agent_id` before applying changes.
- Added mTLS-only agent config polling and acknowledgements, immutable version history, acknowledgement state, and audit events.
- Added rollback by creating a new signed version from historical settings rather than mutating history.
- Added runtime application without process restart and no remote shell/command execution surface.
- Added Connectivity administration controls, signing-key generation tooling, OpenAPI coverage, and remote configuration documentation.

### v8.5 pre-v1 stabilization

- Reduced dashboard backend load by polling only the active workspace and opening satellite/incident SSE streams only while those views are active.
- Added composite PostgreSQL indexes for RF history/anomalies, audit actions, satellite freshness, password-reset expiry, and WebAuthn credential usage.
- Added backup verification, guarded restore tooling, and a full-stack backup/restore drill against an isolated temporary database.
- Expanded authorized load testing with duration mode, authenticated incident/search/satellite cases, and optional result capture.
- Added a sensitive-endpoint audit inventory and dedicated backup/restore operations guide.

### v8.4 WebAuthn, recovery, tracing, and performance

- Added password-plus-WebAuthn authentication for cross-platform USB/NFC security keys with user verification and Redis-backed one-time ceremonies.
- Added password re-authentication for security-key enrollment/removal, atomic authorization-version changes, and session revocation after key-management changes.
- Added opt-in TLS SMTP password recovery with hashed 15-minute single-use tokens, Google Authenticator verification when enabled, and administrator-assisted recovery for WebAuthn-only accounts.
- Added optional OTLP/HTTP OpenTelemetry tracing with W3C trace propagation and response trace IDs.
- Added authenticated global search across satellites, assets, CVEs, and incidents plus targeted search indexes.
- Bounded satellite position calculation with a fixed worker pool and added an authorization-gated load-test helper.
- Added database integration coverage for WebAuthn handles, single-use recovery tokens, and global search.

### v8.3.1 smoke-test compatibility

- Updated the authentication smoke test to exercise the mandatory bootstrap password-change flow before accessing protected session endpoints.
- Kept the production `must_change_password` restriction intact and verified that the bootstrap token is rejected from non-password-management endpoints.
- Continued the Google Authenticator E2E flow with the rotated smoke-test password instead of the insecure bootstrap credential.

### v8.3 identity and authorization hardening

- Added mandatory first-login password change for the bootstrap administrator and database-backed authorization-version enforcement for stale JWTs and role/status changes.
- Hardened password policy to 12-72 bytes with uppercase, lowercase, number, and symbol requirements.
- Added `analyst` and `auditor` roles with explicit operational permissions; auditors can read the audit log and analysts can investigate incidents without operator privileges.
- Added admin-managed account enable/disable and privacy-preserving soft deletion with last-admin and self-lockout protections.
- Added audit events for account lifecycle changes and invalidation of authorization state after role/status changes.
- Added a user-administration guide and updated OpenAPI and roadmap documentation.

### v8.2 observability, sessions, and release integrity

- Added Redis-backed active-session tracking with self-service session listing, per-session revocation, and sign-out-everywhere.
- Added clean Google Authenticator-only 2FA UX, explicit session revocation after 2FA/password changes, and stronger authentication E2E coverage.
- Added authentication outcome and SSE connection Prometheus metrics, provisioned Grafana dashboards, and Prometheus alert rules.
- Added keyless Sigstore Cosign signing and GitHub build-provenance attestations for tagged container releases.
- Updated OpenAPI and operational documentation for session management, Google Authenticator, observability, and signed releases.

### v8.1 pre-v1 security baseline

- Added CycloneDX SBOM generation for all Go modules and the frontend in CI and tagged releases.
- Added dependency review, critical SDR coverage gating, and active SDR fuzzing to CI.
- Added an authentication E2E smoke flow covering login, `/auth/me`, logout revocation, and relogin.
- Added JWT issuer/audience/subject validation, configurable short access-token TTL, and previous-key overlap for controlled signing-key rotation.
- Added user-wide session revocation after password, TOTP, and role changes and fail-closed production Redis startup.
- Added a formal STRIDE threat model, compliance readiness checklist, and software supply-chain guide.
- Hardened forensic replay after fuzzing found non-finite IQ propagation and unbounded capture filenames; forensic files now use restrictive permissions.

### v8 detection engineering

- Added typed RF detection rules with admin-only mutation and viewer-readable configuration.
- Added persisted rule matches, per-rule/per-agent cooldown suppression, and bounded active-rule evaluation.
- Added deterministic correlation that collapses multiple rule matches for one anomaly into one incident using the highest configured severity.
- Added detection-rule and anomaly evidence to correlated incidents and system audit events for automatic incident creation.
- Added Detection Engineering controls to the Signal dashboard and synchronized EN/ES/FR/DE translations and OpenAPI.

### v7.1 hardening

- Enforced incident lifecycle transitions in Go and PostgreSQL.
- Added idempotent incident creation and immutable typed evidence references.
- Added readiness checks and structured request correlation logs without request payloads.
- Reduced default/max pagination to 25/100.
- Replaced SDR TLS verification bypass with optional trusted CA configuration.
- Hardened database backups to avoid credential echoing and use restrictive file permissions.
- Added race detector and pinned gosec security gates to CI.

### Platform

- Go/chi backend with PostgreSQL migrations, Redis integration, Prometheus metrics, graceful shutdown, and request correlation IDs.
- React/Vite dashboard with light/dark themes, EN/ES/FR/DE localization, accessibility improvements, and role-gated analyst/operator/admin workflows.
- Orbital tracking with Celestrak TLE synchronization, SGP4 positions, interactive map, and live SSE updates.
- RF ingestion through `sdr-agent`, authenticated device self-report, IQ capture/replay, and robust anomaly baselines using median/MAD and EWMA techniques.
- TM/TC decoders for AX.25, CCSDS Space Packet, PUS, and CSP v1.
- NVD-backed space CVE synchronization, filtering, keyword discovery, and CVE-to-asset alerts.
- Incident response with RBAC, filtered queues, live SSE updates, investigation timelines, admin-only assignment, and high-confidence RF anomaly escalation.

### Security

- JWT + bcrypt authentication, TOTP, `viewer`/`auditor`/`analyst`/`operator`/`admin` RBAC, admin-managed user creation, and public registration disabled.
- Revocable per-agent API keys; agent self-report identity is derived from credentials.
- Redis-backed token revocation, caching, and distributed rate limiting with separate namespaces for global traffic and login attempts.
- Security headers, request-size limits, strict validation, audit logging, dependency scanning, Trivy image scanning, and production startup checks.
- Integration tests use the production migration chain to prevent schema drift.
- SSE streaming survives middleware instrumentation by preserving `http.Flusher` capability.

### Reliability and CI

- Fixed historical schema drift in integration tests after the `username` migration.
- Fixed satellite SSE integration-test RBAC without weakening production authorization.
- Fixed metrics middleware behavior for SSE and for requests without a Chi route context.
- Fixed a Redis rate-limit key collision that could cause normal dashboard traffic to consume login attempts after sign-out.
- Fixed CVE synchronization writes against the migrated `space_cves` schema.

### Documentation

- Shortened the root README and moved operational detail into focused documents.
- Added architecture and documentation indexes.
- Synchronized API/OpenAPI, deployment, configuration, security, and release documentation with current behavior.

## v9.0

- Added one-time SDR agent enrollment with hashed, expiring tokens.
- Added agent-generated ECDSA keys, CSR signing, client certificates, and mTLS authentication.
- Added automatic certificate rotation, quarantine, irreversible revocation, certificate fingerprints, versions, and last-seen posture.
- Added mTLS identity management to Connectivity while retaining legacy API keys only for migration.
- Added migration `0021_agent_identity_mtls.sql` and SDR Agent CA generation tooling.
