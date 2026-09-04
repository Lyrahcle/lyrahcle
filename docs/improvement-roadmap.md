# Improvement Roadmap

This is the canonical shipped/next/later backlog for the current pre-v1 line. Milestone history lives in `CHANGELOG.md`; security-specific evidence and gates live in `SECURITY_ROADMAP.md`.

Status values: **Shipped**, **Partial**, **Next**, and **Later**.

## Identity, authentication, and authorization

| Improvement | Status | Target |
|---|---|---|
| Google Authenticator TOTP | Shipped | Current |
| Active session inventory and revocation | Shipped | Current |
| Login-specific Redis rate limiting | Shipped | Current |
| Strict input validation | Shipped | Current |
| JWT signing-key rotation overlap and user-wide revocation | Shipped | Current |
| WebAuthn security-key second factor | Shipped | v8.4 |
| Password recovery with TOTP and temporary link | Shipped | v8.4 |
| Force password change on first default-admin login | Shipped | Current |
| Stronger password policy | Shipped | Current |
| Enforce TLS for all production authentication traffic | Shipped | Current |
| CSRF protection if cookie-based authentication is introduced | Shipped | Current (N/A today: bearer-token-only auth, no cookies; `check-csrf-cookie-auth.py` runs in local v1 validation and GitHub Actions, failing the build the moment a handler sets one without CSRF protection) |
| Automatic agent certificate rotation | Shipped | v9.0 |
| Agent enrollment and mTLS | Shipped | v9.0 |
| Additional roles such as analyst and auditor | Shipped | Current |
| Administrator subroles | Shipped | v9 |
| Explicit permission matrix | Shipped | Current |
| Disable and soft-delete users | Shipped | Current |
| Complete sensitive-endpoint authorization review | Shipped | Current (`check-route-authorization.py` audits every route in `cmd/server/*.go` in local v1 validation and GitHub Actions, failing the build if a new route lacks a recognizable role/permission/mTLS/token gate) |
| Complete administrative audit-coverage review | Complete | Before v1.0 |
| Multi-tenant organizations with tenant-scoped RBAC | Shipped | v10.0-v10.6 |
| Database-enforced tenant isolation (PostgreSQL row-level security) | Shipped | v14.2 |
| Platform-administrator control-plane boundary | Shipped | v10.x |
| Granular per-organization RBAC with custom roles and capability enforcement | Shipped | v11.0-v11.1 |
| Detection-rule and operational RBAC capability migration | Shipped | v11.7 |
| Signing-key management abstraction (`KeyProvider`) | Shipped | Current |
| Secret-provider boundary (`env`/`file`, production hardening) | Shipped | Current |

## Detection, RF, and orbital security

| Improvement | Status | Target |
|---|---|---|
| PUS and CSP decoders | Shipped | Current |
| Robust RF anomaly detection | Shipped | Current |
| Typed RF detection rules and deterministic correlation | Shipped | Current |
| Explainable cross-domain correlation | Shipped | v9.4 |
| Interactive correlation constellation and explainable path tracing | Shipped | Block 17 |
| Correlation-driven severity policy after production validation | Later | Before v1.0 |
| RF heat map | Shipped | v14.6.2 |
| Satellite security profile with explainable mission risk and unified evidence | Shipped | Unreleased |
| Evidence provenance and confidence labels; contextual CVE alerts excluded from exposure risk | Shipped | Block 18 |
| Telemetry, RF, CVE, and operations charts | Partial | RF power/spectrum, anomaly timeline, incident severity/timeline, CVE severity/timeline, and orbital altitude charts are shipped; Doppler expected-vs-observed, RF z-score, and fleet agent-status charts remain open |
| Orbital-aware RF detection and TLE freshness gating | Shipped | v9.2/v9.2.2 |
| Physical SDR validation of Doppler/visibility correlation | In progress — RTL-SDR evidence harness delivered; representative lab captures still required | Before auto-escalation/v1.0 |
| Pass prediction and AOS/LOS scheduling | Shipped | v9.3 |
| Maneuver/conjunction context persistence and candidate-only correlation | Shipped | v9.5 |
| Automatic maneuver inference and trusted conjunction screening feeds | Next | v9.x |
| More threat-intelligence sources | Later | v9 |
| SPARTA technique mapping and coverage matrix | Shipped | Unreleased (curated catalog subset; reconcile identifiers upstream before external reporting) |
| Community reception import with geometry consistency check | Shipped | Unreleased (context-only evidence, confidence capped at 60) |
| SPARTA/community coverage workspace | Next | Frontend panel for the two SPARTA and two community-reception endpoints; backend contracts are ready |
| ML-assisted anomaly detection | Later | Post-v1 |
| Advanced/proprietary RF decoders | Later | Enterprise |
| SIEM integrations | Later | Enterprise |

## Performance and scale

| Improvement | Status | Target |
|---|---|---|
| CVE filter indexes | Shipped | Current |
| Response/orbital/CVE caching | Partial | v8.3-v9 (now also covers incidents, asset-alerts, RF readings/anomalies; response caching for other high-traffic reads still open) |
| Satellite-load optimization | Shipped | v8.5 |
| Index review for all filtered searches | Partial | v14.6.4 adds tenant-aware RF/audit/incident hot-path indexes; remaining query families stay under review |
| SQL query optimization | Partial | v14.6.4 removes fleet target N+1 inserts and collapses tenant snapshots to one DB round trip; broader load-driven tuning remains |
| Load-test tooling | Shipped | v8.4 |
| Per-route performance budgets with DB/cache instrumentation | Complete | HTTP and PostgreSQL latency, per-route query counts, rows returned/affected, and cache hit/miss/error/bypass metrics are exported with bounded-cardinality labels; initial alert budgets are documented. True rows examined remain an EXPLAIN/ANALYZE diagnostic rather than a misleading runtime counter. |
| Station federation | Later | v9+ |
| Multi-tenant architecture | Shipped | v10.x (see Identity, authentication, and authorization) |

## Engineering architecture and maintainability

| Improvement | Status | Target |
|---|---|---|
| Testable backend bootstrap/builders | Shipped | Block 12 |
| Reduce `cmd/server/main.go` to composition responsibilities | Shipped | Block 12 |
| Move version-suffixed server files into domain packages | Partial | Next backend architecture iterations |
| Audit error wrapping for `errors.Is`/`errors.As` continuity | Shipped | Block 13 |
| Verify `context.Context` propagation across all stores | Shipped | Block 13 |
| Complete frontend TypeScript migration | Partial | Block 14 onward |
| Frontend CI warning gate covering `console.error`/`console.warn`/unhandled promises | Partial | `check-frontend-test-warnings.py` currently detects only `act(...)` warnings and is not wired into `ci.yml`; needs broader patterns and a CI step that captures Vitest output |
| Product-journey end-to-end tests (Playwright) | Next | Existing E2E coverage (`security_e2e_http_test.go`) is HTTP-level security only; no browser-driven operator-journey suite (login → incident close) exists yet |

## Platform security and supply chain

| Improvement | Status | Target |
|---|---|---|
| CycloneDX SBOM in CI/releases | Shipped | Current |
| STRIDE threat model | Shipped | Current |
| SDR-agent fuzzing | Shipped | Current |
| Dependency and vulnerability gates | Shipped | Current |
| Cosign image signing and build provenance | Shipped | Current |
| Security design review for enterprise modules | Later | Enterprise |
| Compliance-readiness checklist | Shipped | Current |
| Full >80% application coverage | Partial | Before v1.0 |
| Security E2E coverage | Partial | v14.6.3 adds a dedicated HTTP tenant-boundary/IDOR/RBAC/audit gate; broader agent/WebAuthn/remote-config end-to-end matrices remain before v1.0 |
| HashiCorp Vault integration | Later | v9 / production |
| Automated backup and restore drills | Shipped | v8.5 |

## Observability and operations

| Improvement | Status | Target |
|---|---|---|
| Prometheus metrics | Shipped | Current |
| Provisioned Grafana dashboard | Shipped | Current |
| Prometheus alert rules | Shipped | Current |
| Request correlation IDs | Shipped | Current |
| OpenTelemetry distributed tracing | Shipped | v8.4 |
| Automated staging/production CD | Later | v9 |
| Cloud-neutral Kubernetes/Helm deployment foundation | Shipped | v14.6.6 |
| Kubernetes staging/kind smoke path and Prometheus Operator integration | Shipped | v14.6.7 |
| Kubernetes resilience/chaos and local DR evidence | Shipped | Block 16 |
| Kubernetes restricted Pod Security and default-deny network isolation | Shipped | Block 8 |
| AWS EKS infrastructure-as-code and managed-service cutover | Partial | v14.6.8 ships VPC/EKS/ECR/RDS/KMS/Secrets + secure opt-in ElastiCache; LBC/add-on automation, external secret delivery and cutover drills remain |
| Container image scanning | Shipped | Current |
| Signed remote SDR configuration | Shipped | v9.1 |
| IQ forensic capture and playback | Partial | v9 (SigMF 1.0.0 read/write shipped with cf32/ci16/ci8/cu8 decoding, checksum integrity and provenance labelling; annotation-driven forensics still open) |

## UX and documentation

| Improvement | Status | Target |
|---|---|---|
| EN/ES/FR/DE i18n | Shipped | Current |
| Light/dark mode | Shipped | Current |
| Shared toast/snackbar feedback | Shipped | Block 15 |
| Unified empty/loading/error/offline states | Shipped | Block 15 |
| Accessibility baseline | Shipped | Skip navigation, focus visibility, live regions, modal focus trapping/return, Escape semantics, accessible dialog naming and portal-aware axe coverage are enforced; manual assistive-technology validation remains a release gate |
| Header search | Shipped | v8.4 |
| Search across satellites/assets/CVEs/incidents | Shipped | v8.4 |
| OpenAPI and Swagger | Shipped | Current |
| Refactored technical documentation | Shipped | Current |
| Updated README for auth, realtime, and languages | Shipped | Current |
| TLS and secrets deployment guide | Shipped | Current |
| User administration and role guide | Shipped | Current |
| Connectivity/device support guide | Next | v8.3 |
| CVE user guide | Next | v8.3 |
| RF heat-map and chart documentation | Later | With feature |
| Complete user guide PDF | Later | Before v1.0 |
| Changelog maintenance | Shipped | Ongoing |

## Enterprise architecture

| Improvement | Status | Target |
|---|---|---|
| Executive/compliance reports | Later | Enterprise |
| Licensing | Later | Enterprise |
| Shared `rf-lib` | Later | Post-v1 |
| Shared `tle-lib` | Later | Post-v1 |
| Shared UI component library | Later | Post-v1 |
| Fleet policy and staged agent rollouts | Shipped | v14.6.5 |
| Agent hardware capability reporting | Shipped | v13.10 |
| Agent hardware capability physical lab-validation gate | Next | Before v1.0 |
| Stronger remote-config signing-key custody | Next | Before v1.0 |

Note: multi-tenancy moved from "Enterprise architecture / Later" to the core Identity, authentication, and authorization table once it shipped in v10.0; it is core, not an enterprise-only module.

## Current pre-v1 priorities

1. Validate orbital-aware Doppler/visibility correlation with representative physical SDR captures before enabling automatic orbital escalation (currently disabled). The RTL-SDR evidence harness is delivered and public SigMF recordings can now be replayed through the same spectral path; the physical capture campaign and false-positive analysis remain open.
2. Reach and hold >80% coverage on security-critical code paths, including security-focused end-to-end coverage.
3. Exercise multi-replica/AWS resilience and collect DR/RPO/RTO evidence before cutover; backend error-chain and PostgreSQL context-propagation guardrails are now shipped.
4. Complete the agent hardware-capability physical lab-validation gate (RTL-SDR/HackRF) — `compatible` currently reflects self-reported software capability only.
5. Finish remaining operator-facing user guides and perform the final manual assistive-technology/keyboard accessibility validation on the release candidate.
6. Wire the frontend warning gate into CI and widen it beyond `act(...)` to `console.error`/`console.warn`/unhandled promises.
7. Stand up a small Playwright suite (5-8 journeys) covering the core detection-to-incident-closure operator workflow.
8. ✅ Instrument per-route DB query count/latency/rows returned-or-affected and cache hit/miss metrics; document initial performance budgets and retain EXPLAIN/ANALYZE for true rows-examined analysis.

Per-release delivery detail (what shipped in v9.1, v10.x, v11.x, and later) lives in `CHANGELOG.md`, not here, to avoid a second history that can drift out of sync with this backlog.
