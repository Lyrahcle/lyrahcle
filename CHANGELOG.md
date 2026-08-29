# Changelog

Notable changes are tracked here until versioned release notes take over.

## Unreleased

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
- React/Vite dashboard with light/dark themes, EN/ES/FR/DE localization, accessibility improvements, and role-gated operator/admin workflows.
- Orbital tracking with Celestrak TLE synchronization, SGP4 positions, interactive map, and live SSE updates.
- RF ingestion through `sdr-agent`, authenticated device self-report, IQ capture/replay, and robust anomaly baselines using median/MAD and EWMA techniques.
- TM/TC decoders for AX.25, CCSDS Space Packet, PUS, and CSP v1.
- NVD-backed space CVE synchronization, filtering, keyword discovery, and CVE-to-asset alerts.
- Incident response with RBAC, filtered queues, live SSE updates, investigation timelines, admin-only assignment, and high-confidence RF anomaly escalation.

### Security

- JWT + bcrypt authentication, TOTP, `viewer`/`operator`/`admin` RBAC, admin-managed user creation, and public registration disabled.
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
