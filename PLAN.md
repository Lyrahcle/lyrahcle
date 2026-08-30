# Development Plan

Lyrahcle is moving from a collection of space-security capabilities toward a coherent Space Security Operations platform. The detailed shipped/next/later backlog is maintained in `docs/improvement-roadmap.md`.

## Current foundation

- Orbital tracking with live SSE updates
- RF ingestion and statistical anomaly detection
- SDR agent identity and device self-report
- TM/TC decoding
- Space CVE intelligence and asset alerts
- JWT, Google Authenticator TOTP, active session management, RBAC, audit logging, Redis-backed revocation and rate limiting
- EN/ES/FR/DE dashboard localization
- Incident response with timelines, workflow states, typed RF detection rules, cooldown suppression, and deterministic anomaly correlation

## v7.1 hardening baseline

- Strict incident state machine enforced in application and database layers
- Idempotent incident creation and immutable typed evidence references
- Readiness probes and structured request correlation logs
- Safer pagination defaults and CI race/security gates
- SDR TLS verification with optional private CA trust instead of verification bypasses
- Backup output with restrictive permissions and no credential echoing

## Next priorities

### 1. Cross-domain correlation

Extend the shipped RF detection-rule engine across asset alerts, orbital events, agents, and audit evidence while preserving deterministic, explainable incident creation.

### 2. RF signal assurance

Add per-satellite or per-link baselines, replay/spoofing indicators, expected Doppler windows, and stronger signal fingerprints.

### 3. Agent fleet security

v9.0 delivers one-time enrollment, locally generated agent keys, short-lived client certificates, automatic rotation, quarantine, revocation, and direct mTLS authentication. v9.1 adds signed, declarative remote capture configuration with acknowledgements and rollback. Next work is fleet policy, staged rollout, and hardware capability constraints.

### 4. Orbital security context

Add pass prediction, expected Doppler windows, visibility checks, maneuver/anomaly signals, and conjunction context.

### 5. Asset graph and exposure

Model relationships between programs, satellites, ground stations, radios, firmware, agents, users, vulnerabilities, and incidents. Use those relationships to prioritize attack paths and exposure.

### 6. Production hardening

Complete real SDR hardware validation, high-availability deployment guidance, distributed tracing, backup/restore drills, and operational runbooks. Prometheus/Grafana observability, signed container images, build provenance, CycloneDX SBOM generation, dependency review, SDR fuzzing, and JWT rotation overlap are now part of the pre-v1 baseline.

## v1.0 gate

v1.0 should require green CI, production-safe defaults, tested recovery procedures, real SDR validation, documented trust boundaries, and a stable operator workflow for detection through investigation.

## Identity and authorization hardening

Delivered in v8.3: bootstrap password rotation, stronger password policy, analyst/auditor roles, account disable/delete, and a documented permission matrix. WebAuthn security-key authentication and opt-in password recovery are delivered in v8.4. Hardware-backed account recovery remains deliberately administrator-assisted when no TOTP factor exists.

## v8.4 identity and performance milestone

Delivered: cross-platform WebAuthn security keys, opt-in TLS SMTP password recovery, OTLP tracing export, authenticated global search, targeted search indexes, bounded orbital-position workers, and a controlled load-test helper.

Next: broader endpoint audit coverage, database query plans under production-like data, richer operational charts, connectivity/CVE operator guides, restore drills, and agent certificate enrollment.

## v9.0 agent trust milestone

Delivered: dedicated SDR Agent CA support, one-time enrollment tokens, CSR signing, certificate-backed identity, automatic pre-expiry rotation, quarantine, irreversible revocation, version posture, and Connectivity UI lifecycle controls. Legacy API keys remain as a migration path only.


## v9.1 remote management milestone

Delivered: Ed25519-signed immutable configuration versions, mTLS-only polling, runtime frequency/interval changes, applied/rejected acknowledgements, rollback as a new signed version, audit coverage, and Connectivity controls. Remote management deliberately excludes shell execution and arbitrary command payloads.
