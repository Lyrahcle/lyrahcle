# Unreleased
- Added RF expected-vs-observed Doppler analysis: a protected `GET /api/v1/rf/doppler` endpoint derives orbital Doppler from the existing TLE/ground-station model for recent satellite-linked frequency anomalies, returns expected frequency, signed residual and tolerance, and a localized RF workspace panel renders only real available comparisons without synthesizing missing telemetry.
- Rebuilt the Correlation Constellation Orbital Focus on a real WebGL renderer using `three`, `@react-three/fiber` (8.x for React 18) and `@react-three/drei`, replacing the in-house SVG perspective projection. The scene now has genuine depth, a damped orbital camera with smooth focus easing, distinct geometry per node kind (satellite orbs with rings, incident octahedra, thin hexagonal CVE plates, ground-station pillars with dishes, asset and program prisms), thin additive evidence curves merged into a single line geometry, and an animated explainable route as the dominant visual element.
- Added `src/lib/correlationLayout3d.ts`, a renderer-independent 3D layout engine: breadth-first orbital shells, azimuth-sector clustering by node kind, and a deterministic force relaxation pass (edge attraction, minimum-separation repulsion, shell reprojection). The layout is seeded from node identifiers rather than randomness, stays readable with zero links, and tightens genuinely connected evidence when links are dense. Covered by 11 unit tests.
- Reworked the workspace UX so the scene owns the viewport: a compact HUD rail with a single filter popover replaces the permanent control blocks, the context panel is narrower and denser, search gained autocomplete with arrow-key and Enter navigation, and a new focus mode isolates the selected node's neighbourhood and traced route.
- Kept the correlation model, permission handling, trust boundaries and explainable-route semantics unchanged; the rebuild is presentation and interaction only, with no backend route, schema migration or graph runtime dependency.
- Made the constellation usable without WebGL and without a mouse: the node index is always present in the DOM as focusable buttons carrying kind and label, visually hidden below the extra-large breakpoint and shown as a rail above it, and an explicit fallback message replaces the canvas when no WebGL context exists. Node spin, gentle orbit and route animation respect `prefers-reduced-motion`.
- Added the constellation polish pass: optional per-kind cluster markers driven by the layout centroids, numbered hop markers along a traced route with everything outside the route dimmed, hover cursor feedback on scene nodes, and disposal of the merged link geometries when the evidence fabric changes.
- Added a polite live region announcing the current orbital focus, its kind and its connection count whenever the selection changes, localized in EN/ES/FR/DE.
- Isolated three.js into a lazily loaded `three-vendor` chunk via rolldown `advancedChunks` so it stays out of the initial document payload, and gave `scripts/check-frontend-bundle.py` a documented raw-size exemption for it while leaving the gzip budget unchanged.
- Rewrote `scripts/check-correlation-graph.py` for the new architecture (49 checks covering the declared 3D runtime, React 18 compatible fiber major, removal of the legacy SVG renderer, code splitting, layout determinism and renderer independence, keyboard reachability, non-WebGL fallback and localized HUD copy) and extended its meta-test suite from 4 to 19 cases.
- Reworked Correlation Constellation into a viewport-first minimal 3D workspace and preserved disconnected mission nodes as visible spatial context.

- Added Block 17 Correlation Constellation as a dedicated flagship investigation workspace with a minimal 3D orbital-focus renderer, draggable camera, mathematical perspective/depth ordering, deterministic cross-domain relationships, time/type/severity filtering, keyboard-accessible SVG nodes, exact-match exposure context, shortest explainable path tracing and direct navigation into satellite, station, RF, incident, asset and CVE workspaces. The renderer is implemented in Lyrahcle with no new graph runtime dependency or backend/schema change.
- Added Block 16 resilience and disaster-recovery validation with local kind pod-recovery, PDB/node-drain, Redis/PostgreSQL connectivity-fault drills and machine-readable RTO evidence.
- Added a non-destructive local PostgreSQL restore drill using `lyrahcle_test`, a temporary restore database, exact user-table row-count comparison and cleanup on every exit path.
- Hardened production Helm topology spreading with strict zone/hostname scheduling, minimum domains and fail-closed PDB/topology validation.
- Made CVE sync replica count honor Helm values and documented AWS RTO/RPO objectives as design targets that remain unverified until a production-like restore/failover exercise.

# Changelog
- Migrated the frontend shared library layer to TypeScript with local and CI type checking while preserving the existing UI casing conventions.

## Unreleased
- Fixed Tailwind source discovery to include TypeScript and TSX so the correlation 3D stage and other migrated frontend classes are emitted in production CSS.

- Added a shared frontend UX foundation with global toast/snackbar feedback, reusable empty/loading/error/offline/forbidden states, retryable rate-limit loading, unified empty states for incidents/CVEs/asset alerts/RBAC, a keyboard skip link, consistent focus-visible styling and EN/ES/FR/DE copy. Success/info toasts auto-dismiss after six seconds while warning/error feedback remains persistent; no backend API, database schema or runtime dependency changes.
- Continued the frontend TypeScript migration through the application boundary and security surfaces: migrated `App`, `Dashboard`, `Panel`, RBAC, rate-limit, secret-provider and incident-response components to TSX; extracted hash routing into a typed hook and permission-aware dashboard destinations into a typed navigation model; removed two version-suffixed frontend test filenames; added explicit JS interop boundaries for not-yet-migrated JSX children so TypeScript does not infer invalid `never[]` or callback prop contracts; and added a phase-2 migration contract. No API, schema or intentional visual changes.
- Hardened backend reliability around causal errors and cancellation: audited all 166 production `fmt.Errorf` calls, added a regression contract that rejects error arguments formatted without `%w`, made redacted Redis connection failures safely classifiable with `errors.Is`, verified every PostgreSQL-backed store method propagates caller `context.Context`, and bounded pool cleanup plus migration advisory-lock release to two seconds. No API, schema or UI changes.
- Refactored backend composition into `internal/bootstrap` builders for repositories, Redis, agent signing/CA, WebAuthn, rate limiting, request services and HTTP server construction. Reduced `cmd/server/main.go` from 1,041 to roughly 310 lines by separating HTTP helpers, mission runtime logic, validation/audit helpers and startup seeding without changing routes, schemas or runtime dependencies. Added bootstrap unit coverage and a structural regression contract.
- Added AWS production-readiness wiring with optional Route53-validated ACM certificates, ExternalDNS on a dedicated Route53-scoped EKS Pod Identity role, immutable ECR build/push tooling that resolves image digests, dynamic non-secret Helm release-value rendering, and a read-only AWS readiness check. The AWS root still does not create hosted zones, secret values or the Lyrahcle application Helm release automatically.
- Automated AWS Load Balancer Controller installation for EKS with a dedicated least-privilege IAM policy, EKS Pod Identity association, pinned Helm provider/chart versions, two-replica production guardrail, ALB ACM/HTTPS example annotations and static regression contracts. ALB/NLB lifecycle remains Kubernetes-reconciled from Lyrahcle Ingress/Service resources; no application runtime or database schema changes.
- Added a permission-aware command palette opened with Ctrl/Cmd+K, combining fast section navigation with debounced global search across satellites, assets, CVEs and incidents. Keyboard navigation supports arrows, Enter and Escape, backend results are filtered to destinations available in the current UI, and satellite search results open the selected Satellite Security Profile directly. Added EN/ES/FR/DE copy, focused frontend tests and a command-palette regression contract. No database schema changes and no new runtime dependencies.
- Replaced the top-left dropdown navigation with a permission-aware application sidebar grouped into Overview, Mission, Security, RF operations, Platform and Preferences. The sidebar is persistent and collapsible on desktop, becomes an accessible mobile drawer, preserves hash deep links and permission-gated destinations, and moves the active workspace context into the header. Added EN/ES/FR/DE copy, focused frontend tests and a navigation-shell regression contract. No database schema changes and no new runtime dependencies.
- Added Ground Station Operations as a permission-aware ground-segment workspace for configured SDR station endpoints, with station mapping, online/attention/offline posture, explainable station risk, current satellite visibility from propagated positions, 24-hour observed-satellite/RF context, correlated incidents and RF anomalies, and direct navigation to RF, incidents and fleet management. Added shared geometry/risk helpers, EN/ES/FR/DE copy, frontend tests and a station-isolation contract. No database schema changes and no new runtime dependencies.
- Added a shared, explainable 0–100 risk engine used by mission operations, satellite security profiles and the satellite inventory; risk factors are bounded and grouped by operational domain. Standardized UI copy to sentence case while preserving technical acronyms, removed forced all-uppercase presentation, and cleaned ordinary source/configuration comments with regression contracts to keep the codebase visually clean. No database schema changes and no new runtime dependencies.
- Added a satellite security profile opened directly from orbital management, with an explainable 0–100 mission risk score, satellite-scoped incident retrieval, RF anomaly correlation, seven-day orbital-event context, related SDR agents, conservative exact-match asset exposure context, and a unified evidence timeline; localized in EN/ES/FR/DE and covered by focused frontend/static contracts.
- Hardened local Docker startup against transient Docker Hub DNS/registry outages: local services now have stable image tags, unnecessary `docker/dockerfile:1` frontend lookups were removed, and `scripts/docker-local-up.sh` falls back to previously built local images with `--no-build --pull never` when registry resolution fails, including automatic reuse/tagging of the legacy `lyrahcle-main-*` Compose images from existing development checkouts. Missing uncached base images are reported explicitly because host/WSL DNS failures cannot be repaired by application code.
- Added the Mission Security Operations dashboard to the default orbital landing page, combining permission-aware incident, RF anomaly, SDR fleet, orbital freshness and asset-exposure posture into one operator view with a unified activity timeline, actionable queue and cross-domain quick navigation. The implementation reuses existing APIs, adds no schema/runtime dependencies, and includes EN/ES/FR/DE localization plus focused frontend regression coverage.
- Raised test coverage on two security-critical backend packages toward the pre-v1.0 >80% goal: `internal/config` (54.8% → 63.9%) by covering `loadSecretProvider`'s file/remote-provider selection paths, token-file reading, and `getSecret`'s not-found fallback; `internal/auth` (81.1% → 87.2%) by covering `RequireAnyRoleWithFallback`'s allowlist enforcement and `RequireAgentKeyOrRoleWithFallback`'s agent-key/current-secret/previous-secret/reject paths, which previously had zero test coverage despite gating authenticated routes. Full `go test ./...` and `go vet ./...` pass with no changes to production code.
- Fixed 11 broken documentation links in `docs/README.md` left over from the browser-security/RBAC/multi-tenancy/SDR-agent doc consolidation: `browser-auth-security.md`, `browser-and-cors-security.md`, `rbac.md`, `rbac-endpoint-matrix.md`, `detection-rbac-v117.md`, `multi-tenancy.md`, `postgres-row-level-security.md`, `tenant-isolation-audit.md`, `tenant-runtime-integrity.md`, `agent-hardware-capabilities.md` and `documentation-maintenance.md` no longer existed on disk but were still linked; the index now points to `01-browser-security-guide.md`, `02-rbac-guide.md`, `03-multi-tenancy-guide.md` and `04-sdr-agent-guide.md`, and the obsolete `documentation-maintenance.md` row was removed.
- Deleted `docs/agent-enrollment.md`, `docs/agent-remote-configuration.md` and `docs/00-CONSOLIDATION_PLAN.md`, whose content is fully superseded by `docs/04-sdr-agent-guide.md`; updated the remaining cross-references in `README.md`, `docs/configuration.md`, `docs/api-reference.md`, `docs/deployment.md` and `docs/incident-response-runbook.md` to point at the consolidated guide.
- Hardened `Dockerfile.cve-sync` to run as a dedicated non-root user (uid/gid 65532), matching the backend, frontend and sdr-agent images; previously this was the only container image without an explicit `USER`, so it ran as root unless the orchestrator's `securityContext` overrode it (Helm already enforced `runAsNonRoot`, but standalone `docker run`/local `docker-compose` use did not).

## v14.6.8 — AWS infrastructure-as-code foundation

- Fixed the non-root frontend container startup on Docker Compose and Kubernetes by making `/run` writable to nginx UID 101 and mounting the Kubernetes runtime `emptyDir` directly at `/run`; added a regression contract so `/var/run` symlink ownership cannot reintroduce the CI failure.
- Added a Terraform/OpenTofu AWS root module for a future Lyrahcle migration with a multi-AZ VPC, private EKS managed node groups, ECR, encrypted RDS PostgreSQL, KMS and Secrets Manager.
- Added production guardrails for three-AZ topology, restricted EKS public API CIDRs, minimum two-node capacity, one NAT per AZ, RDS Multi-AZ/deletion protection/final snapshots and longer backups.
- Added immutable scan-on-push ECR repositories for backend, frontend and CVE-sync images.
- Added optional ElastiCache/Valkey infrastructure with KMS at-rest encryption, TLS in transit, Multi-AZ failover and production AUTH enforcement; it stays disabled by default until first-class Lyrahcle Redis TLS/auth support ships.
- Added Terraform/OpenTofu local planning tooling, static IaC contracts and a dedicated GitHub Actions Terraform format/init/validate workflow.
- Bumped the Helm chart to 0.3.0 / appVersion 14.6.8 and kept AWS load balancers in the Kubernetes reconciliation layer rather than provisioning ALB/NLB directly in Terraform.
- Removed hardcoded release numbers from Kubernetes static contracts; they now validate semantic `appVersion` and production image-tag coherence so normal chart version bumps do not break unrelated deployment gates.
- Added AWS infrastructure documentation and remote-state guidance; application secret values remain out of Terraform state by creating Secrets Manager metadata only.
- No database schema changes and no new application runtime dependencies.

## v14.6.7 — Kubernetes staging and observability

- Added a portable two-replica staging Helm overlay plus a three-node kind topology and disposable PostgreSQL/Redis manifests for local Kubernetes lifecycle testing.
- Added `k8s-local-up.sh`, `k8s-smoke-test.sh` and `k8s-local-down.sh` to build/load local images, install the chart, assert two ready backend/frontend replicas, and verify `/readyz` plus backend/CVE Prometheus metrics.
- Added a dedicated CVE-sync metrics Service, optional Prometheus scrape annotations, Prometheus Operator `ServiceMonitor` resources and an optional `PrometheusRule` with application/security/worker alerts.
- Backend HTTPS monitoring fails Helm validation unless ServiceMonitor scraping has a CA Secret or an explicit insecure verification override; production/AWS examples keep certificate verification enabled.
- Extended NetworkPolicy with an opt-in monitoring namespace/pod path so metrics scraping remains compatible with backend ingress isolation.
- Extended local/CI Helm gates to lint/render the staging overlay and render ServiceMonitor/PrometheusRule resources.
- Added Kubernetes staging/observability documentation and kept AWS-specific collectors, IAM and remote-write configuration outside the cloud-neutral application chart.
- No database schema changes and no new application runtime dependencies.

- Refactored backend API route composition out of the monolithic server bootstrap into domain-focused modules while preserving route registration order and authorization contracts.
- Centralized security and UI/API parity validation in `scripts/check-contracts.sh`, and made source-level contract checks scan the modular server tree instead of depending on `main.go`.
- Hardened frontend API streaming with shared SSE parsing, consistent endpoint metadata on network/auth errors, safe alert-ID URL encoding and removal of duplicate satellite update handling.
- Aligned contributor documentation with the Go 1.26.7 toolchain, fixed shell examples whose working-directory state leaked between commands, and excluded Python cache artifacts from source archives.
- Added explicit JSON content type coverage for `/healthz`. No database schema changes.
- Made the dashboard's active section deep-linkable via a lightweight hash-based route (`#/orbital`, `#/admin`, ...), enabling bookmarking and browser back/forward without adding a router dependency or requiring SPA-fallback server configuration.








## v14.6.6 — Kubernetes deployment and future EKS foundation

- Added a cloud-neutral Helm chart for backend, frontend and CVE sync with rolling updates, probes, resource controls, optional HPA/PDB/topology spreading, Ingress, direct agent LoadBalancer service and opt-in NetworkPolicy.
- Added separate backend/frontend/CVE-sync ServiceAccounts with Kubernetes API token automount disabled, preserving a least-privilege path for future workload identity such as EKS Pod Identity.
- Added file-secret projection hardening: Kubernetes Secret files are copied into an in-memory runtime volume with strict `0400` permissions so Lyrahcle's hardened file secret provider does not accept Kubernetes symlink projections directly.
- Added production Helm validation that rejects plaintext backend service, mutable `latest` image tags, backend TLS without frontend verification, and direct agent exposure without mTLS.
- Added generic production and AWS/EKS example values. The AWS mapping uses ALB IP targets for the UI and an optional NLB `LoadBalancerClass` with IP targets/TLS passthrough for SDR-agent mTLS.
- Serialized PostgreSQL schema migrations with a session advisory lock and added a concurrent multi-replica migration regression test so multiple backend Pods may start safely against the same database.
- Added Kubernetes deployment and future EKS migration documentation, including the explicit Redis TLS gap that must be closed before a managed encrypted Redis production cutover.
- Added a Kubernetes deployment contract to local validation and Helm lint/render checks to GitHub Actions.
- Updated the frontend nginx image/config to support runtime backend upstream/TLS settings and a dedicated `/healthz` probe endpoint.
- No database schema migration and no new application runtime dependency.

## v14.6.5 — Fleet staged-rollout policy and ACK deadline hardening

- Enforced staged-rollout target eligibility against the latest tenant-scoped SDR hardware report: active status, fresh `compatible` RX posture, unchanged integrity baseline, and requested-frequency coverage are now required at creation and revalidated before every batch.
- Added operator-visible fleet-policy eligibility in Connectivity so incompatible, stale, tampered, missing-report, and out-of-range agents are shown as blocked and cannot be selected for a staged rollout.
- Added configurable `ack_timeout_seconds` (30-86400 seconds, default 900) and a partial issued-target deadline index; promotion fails closed when an issued target exceeds the acknowledgement deadline.
- Added an ACK deadline/countdown to the rollout UI and a `Fail overdue batch` action that reconciles the server state to `failed` through the existing guarded rollout action path.
- Aligned the UI rejection-rate calculation with the backend safety rule by using only decided targets (`applied + rejected`) rather than the full pending cohort.
- Added centralized `agent.rollout_failed` auditing for rejection-threshold, acknowledgement-timeout, target-policy, and configuration-issuance failures.
- Added PostgreSQL-backed fleet eligibility/deadline regression coverage, frontend policy/deadline tests, and an explicit v14.6.5 fleet hardening gate to `scripts/test-local-v1.sh`.
- Adds migration `0042_fleet_rollout_hardening.sql`; no new runtime dependencies.

## v14.6.4 — Database hot-path and fleet performance hardening

- Replaced per-agent fleet-rollout target insertion with one set-based `unnest(... WITH ORDINALITY)` statement, preserving caller order while keeping the entire rollout transaction atomic when an agent is missing or inactive.
- Collapsed tenant-isolation posture snapshots from nine sequential count queries into one PostgreSQL round trip.
- Added tenant-aware hot-path indexes for RF history by agent, recent audit evidence by actor, and incident filtering by agent, assignee, and satellite.
- Tightened fleet acknowledgement refreshes to join `agent_config_acks` on `organization_id`, matching the tenant-aware index and avoiding unnecessary cross-tenant candidates.
- Added migration/implementation regression coverage plus an explicit PostgreSQL-backed performance hardening gate to `scripts/test-local-v1.sh`.
- Adds migration `0041_performance_hot_paths.sql`; no new runtime dependencies.

## v14.6.3 — HTTP security E2E and tenant-boundary hardening

- Added a dedicated HTTP-level security E2E gate covering tenant-isolated asset lists, cross-tenant asset/incident/user IDOR attempts with valid foreign UUIDs, foreign parent references, viewer-vs-manage RBAC, forged organization claims and auth-version token invalidation.
- Added tenant-scoped audit E2E coverage proving one organization cannot read another organization's login events.
- Fixed successful login auditing to bind the event explicitly to the authenticated user's organization; login executes before authentication middleware attaches tenant claims, so the previous generic audit helper incorrectly attributed non-default-tenant logins to the default organization.
- Wired `TestSecurityE2E*` into `scripts/test-local-v1.sh` as an explicit PostgreSQL-backed release gate in addition to the normal server integration suite.
- No database schema changes and no new runtime dependencies.

## v14.6.2 — RF signal heat map and local integration preflight

- Replaced the signal dashboard's coordinate-only station coverage panel with an operator-facing RF heat map driven by captured `power_dbm`, `frequency_hz`, `captured_at` and configured station coordinates.
- Added agent, frequency-range and time-window filters, per-station average/peak power details, accessible markers, a calibrated -120 to -30 dBm intensity legend and explicit no-interpolation semantics so the UI does not imply unmeasured geographic coverage.
- Added English, Spanish, French and German RF heat-map copy plus frontend regression coverage for aggregation, filtering, invalid ranges and missing coordinates.
- Hardened the local v1 runner with a real PostgreSQL connection/database preflight and made Redis integration tests retain the configured PostgreSQL URL, allowing mixed Redis+database alert tests to execute instead of being skipped when both services are available.
- No database schema changes and no new runtime dependencies.

## v14.6.1 — WebAuthn ceremony and secret-provider coverage hardening

- Hardened WebAuthn ceremony consumption so purpose/user validation happens before the one-time Redis delete; a mismatched purpose can no longer destroy a legitimate ceremony, while an atomic compare-and-delete still permits only one successful consumer.
- Expanded `internal/secretprovider` regression coverage across environment/file/remote custody, including path/key validation, symlink and permission defenses, remote HTTPS/status/payload failures, bearer authentication and health checks.
- Expanded `internal/correlation` boundary coverage for confidence clamps, severity ordering and domain deduplication.
- Added >=80% local coverage gates for `internal/secretprovider` and `internal/correlation`, plus a Redis-backed >=80% gate for `internal/webauthnflow` when `TEST_REDIS_ADDR` is configured.
- No database schema changes.

## v14.6.0 — Scheduler and singleton-job HA hardening

- Added token-guarded Redis lease heartbeats so long-running singleton jobs retain their lease beyond the original TTL while renewal succeeds on the owning backend replica.
- Lease renewal stops if ownership is lost, and stale owners still cannot remove a replacement lease.
- Hardened the scheduler with panic recovery, same-replica overlap prevention, explicit job validation and graceful stop waiting for active jobs.
- Added >=80% local scheduler coverage and a Redis-backed >=80% job-lock coverage gate when `TEST_REDIS_ADDR` is available.
- No database schema changes.

## v14.5.9 — Agent identity coverage completion

- Added CA/parser coverage for legacy EC and PKCS#1 private-key formats, missing CA files, non-signing CA certificates, accepted/rejected agent public-key classes, malformed public-key comparisons, RSA-2048 client certificates and non-positive certificate TTLs.
- Keeps the critical `internal/agentidentity` coverage gate at >=80%; the threshold is not relaxed.
- No runtime or database schema changes.

## v14.5.8 — Key-provider coverage completion

- Added lifecycle/error-path coverage for nil providers, bootstrap imports, managed activation/revocation failures, staged rotation conflicts, public-key export verification, generated key-pair round trips and malformed managed manifests.
- Keeps the critical `internal/keyprovider` coverage gate at >=80%; the threshold is not relaxed.
- No runtime or database schema changes.

## v14.5.7 — Tenant rate-limit and scope hardening

- Expanded tenant-aware rate-limit regression coverage for invalid policy budgets, identity/class validation, privacy-preserving bucket keys, Redis-outage fallback behavior and cross-instance tenant isolation.
- Expanded `dbscope` coverage to assert fail-closed defaults and ensure request, tenant and platform-control-plane markers remain explicit and independent.
- Added local >=80% coverage gates for `internal/dbscope` and, when Redis is available, `internal/ratelimit`.
- Documented through tests that Redis loss deliberately degrades the distributed limiter to a bounded per-process fallback; HA work should account for this reduced global enforcement.
- No database schema changes.

## v14.5.6 — Session and distributed revocation hardening

- Hardened `sessionstore.Delete` so a session ID cannot be deleted through a mismatched user scope; cross-user attempts are reported as not found and leave the target session intact.
- Added regression coverage for invalid session registration, Redis outages, corrupt session payloads, stale/foreign list members, revocation key isolation, corrupt user cutoffs and inclusive user-revocation boundaries.
- Added conditional >=80% local coverage gates for `internal/sessionstore` and `internal/tokenblocklist` when `TEST_REDIS_ADDR` is configured.
- No database schema changes.

## v14.5.5 — Agent trust-boundary hardening

- Expanded certificate-authority tests around malformed CA material, mismatched keys, weak CSR keys, SPIFFE identity binding, TTL capping and CA-expiry refusal.
- Expanded remote agent-config signing tests around provider failures, algorithm/KID validation, signing failures and malformed Ed25519 signatures.
- Added >=80% local coverage gates for `internal/agentidentity` and `internal/agentconfig`.
- No database schema changes.

## v14.5.4 — Authentication coverage hardening

- Expanded `internal/auth` regression coverage for token secret rotation, signing-method confusion, malformed claims, token revocation failures, forced-password-change paths, agent certificate precedence and capability-check failures.
- Added an explicit >=80% authentication/authorization coverage gate to `scripts/test-local-v1.sh`.
- No runtime behavior or database schema changes.

## v14.5.3 — Key-management coverage hardening

- Expanded `internal/keyprovider` regression coverage around malformed/symlink key material, managed-manifest path traversal, grace expiry, KID validation, remote signer failures and rotation-policy validation.
- Added an explicit >=80% key-provider coverage gate to `scripts/test-local-v1.sh`.
- No runtime behavior or database schema changes.

Notable changes are tracked here until versioned release notes take over.

## Unreleased

- Added a shared frontend UX foundation with global toast/snackbar feedback, reusable empty/loading/error/offline/forbidden states, retryable rate-limit loading, unified empty states for incidents/CVEs/asset alerts/RBAC, a keyboard skip link, consistent focus-visible styling and EN/ES/FR/DE copy. Success/info toasts auto-dismiss after six seconds while warning/error feedback remains persistent; no backend API, database schema or runtime dependency changes.
- Continued the frontend TypeScript migration through the application boundary and security surfaces: migrated `App`, `Dashboard`, `Panel`, RBAC, rate-limit, secret-provider and incident-response components to TSX; extracted hash routing into a typed hook and permission-aware dashboard destinations into a typed navigation model; removed two version-suffixed frontend test filenames; and added a phase-2 migration contract. No API, schema or intentional visual changes.
- Added AWS production-readiness wiring with optional Route53-validated ACM certificates, ExternalDNS on a dedicated Route53-scoped EKS Pod Identity role, immutable ECR build/push tooling that resolves image digests, dynamic non-secret Helm release-value rendering, and a read-only AWS readiness check. The AWS root still does not create hosted zones, secret values or the Lyrahcle application Helm release automatically.
- Automated AWS Load Balancer Controller installation for EKS with a dedicated least-privilege IAM policy, EKS Pod Identity association, pinned Helm provider/chart versions, two-replica production guardrail, ALB ACM/HTTPS example annotations and static regression contracts. ALB/NLB lifecycle remains Kubernetes-reconciled from Lyrahcle Ingress/Service resources; no application runtime or database schema changes.
- Added Ground Station Operations as a permission-aware ground-segment workspace for configured SDR station endpoints, with station mapping, online/attention/offline posture, explainable station risk, current satellite visibility from propagated positions, 24-hour observed-satellite/RF context, correlated incidents and RF anomalies, and direct navigation to RF, incidents and fleet management. Added shared geometry/risk helpers, EN/ES/FR/DE copy, frontend tests and a station-isolation contract. No database schema changes and no new runtime dependencies.
- Added a shared, explainable 0–100 risk engine used by mission operations, satellite security profiles and the satellite inventory; risk factors are bounded and grouped by operational domain. Standardized UI copy to sentence case while preserving technical acronyms, removed forced all-uppercase presentation, and cleaned ordinary source/configuration comments with regression contracts to keep the codebase visually clean. No database schema changes and no new runtime dependencies.
- Added a Platform health overview with backend dependency readiness, secret-provider health, signing-custody posture, organization count, and platform-admin count. The overview reuses `/readyz` plus existing protected platform security endpoints and does not expose secret material.
- Added a unified Connectivity fleet posture view with healthy/attention/offline counts, agent search and posture filters. Surfaces last contact, hardware compatibility and edge-integrity state before the detailed identity/config/hardware panels.
- Added an operational incident posture summary with active, urgent and unassigned counts for the visible page, quick filters for new/investigating/critical incidents plus a one-click filter reset, compact incident context metadata, a visual event timeline in the selected-incident detail, and frontend coverage for posture counters and quick-filter behavior.
- Closed out the pre-v1.0 sensitive-endpoint authorization review with `scripts/check-route-authorization.py`, a CI contract that audits every route in `cmd/server/*.go` and fails the build if a new route lacks a recognizable role/permission/mTLS/enrollment-token gate. Closed out the CSRF review with `scripts/check-csrf-cookie-auth.py`, which fails the build the moment any handler sets a cookie without CSRF protection (today's auth is bearer-token only, so no cookie is set and CSRF does not apply). Both are wired into `scripts/test-local-v1.sh`.
- Fixed a duplicate `rel` attribute on the incident-response runbook link (`IncidentResponsePanel.jsx`) that silently dropped `noopener` at runtime despite appearing present in the source; `check-frontend-xss.py` now flags any `target=_blank` link with more than one `rel` attribute instead of only inspecting the first one.
- Gave the account menu (`AccountMenu.jsx`) the same `role="menu"`/`role="menuitem"` ARIA structure already used by the primary navigation menu, with test coverage for the menu contract and Escape-to-close behavior.
- Removed `Tabs.jsx`, dead code left over from the pre-v11.16 horizontal-tabs navigation that the top-left dropdown menu replaced.
- Closed an i18n coverage gap: `NavigationMenu`, `AuditLogPanel`, `AnomaliesPanel`, `RfPanel`, `AgentKeysPanel` and `SecretProviderPanel` had hardcoded English UI text (table headers, empty states, button labels, the primary navigation menu chrome) despite the shipped EN/ES/FR/DE i18n. All six now use `useI18n`/`t()` with parity-checked translations in all four languages. `METRIC_LABELS` in `AnomaliesPanel` and the incident-response playbook prose in `IncidentResponsePanel` remain English-only and are a follow-up.
- Added short-TTL (3s) response caching for `GET /incidents`, `GET /asset-alerts`, `GET /rf/readings` and `GET /rf/anomalies`, tenant-isolated by cache key, with explicit invalidation of the default (unfiltered, first-page) view on create/update/acknowledge/ingest. Cuts repeated Postgres load from the frontend's 8s dashboard poll under concurrent operators. Filtered/paginated views beyond the default rely on the TTL, not explicit invalidation, to bound staleness — documented in `response_cache_v143_test.go`.
## v14.3 security hardening

- Added a centralized backend secret-provider abstraction (`env`, hardened file, HTTPS remote) with production fail-closed behavior for process-environment custody.
- Added Platform UI visibility for secret-provider health/metadata without exposing secret values.
- Added an explicit post-authorization platform control-plane DB scope so global Platform operations remain compatible with PostgreSQL RLS without granting tenant users a bypass.
- Added SDR edge-node integrity measurements with backend-owned baselines and UI tamper status.
- Added a Lyrahcle incident-response runbook covering privileged-account, signing-key, SDR-agent, edge-tamper, cross-tenant, secret, database, evidence-preservation, recovery, and post-incident procedures, with a Platform quick-action panel.
- Added the SDR hardware/firmware supply-chain threat model and hardened security account dialogs with viewport portals and responsive modal layout.

## v14.2 tenant database isolation

- Added PostgreSQL `FORCE ROW LEVEL SECURITY` for core tenant-owned operational tables.
- Added request/organization propagation into pgx pooled sessions with mandatory reset on release.
- HTTP database access without an organization now fails closed at the database policy layer.
- Added parent-derived RLS for incident events and asset CVE alerts.

## v11.16

- Fixed PostgreSQL asset projection/scan parity exposed by CI integration tests and completed capability-based auth for hybrid RF ingest, detection rules, RF reads, alerts, and machine-only agent self-service routes.
- UI: hardened Fleet staged rollouts with cohort progress, per-agent states, soak countdown, rejection guardrails, and rollback confirmation.
- Fixed the local CI gate to invoke the script-permission contract through Python and hardened the Orbital Dashboard KPI test for duplicate ratios.
- Added Orbital Dashboard v2 with constellation KPIs, TLE freshness, operator attention queue, recent updates, and quick navigation to orbital operations.
- Hardened local/CI validation: restored executable script modes, added a script-permission contract, and fixed the incident-response external-link noopener contract.
- Fixed the top-left navigation dropdown to use an opaque background for consistent readability.
- Replaced the horizontal primary tabs with a top-left dropdown navigation menu; Orbital is the default landing dashboard and remains reachable from the Lyrahcle logo; navigation entries are permission-aware and expose Admin/Platform only when authorized; added Settings to the primary page menu and active-page indication with keyboard/Escape/outside-click behavior.
- Fixed: rendered Active Sessions through the same viewport portal shell used by Change Password and Google Authenticator, avoiding clipping/stacking inside the account menu.
- Fixed: made navigation contract tests portable under Vitest/jsdom by resolving source files from `process.cwd()` instead of `import.meta.url`.

## v11.1

- Enforced granular capabilities across sensitive HTTP read/write surfaces.

## v11.0.1

- Fixed backend import paths to match the Go module declared in `apps/backend/go.mod`.

## v11.0

- Granular organization-scoped RBAC foundation with permissions, custom roles, compatibility mappings and permission middleware.

## v10.6

- Added tenant-isolation audit gates, full tenant snapshots, source-contract checks and optional database-backed Org-A/Org-B integration tests.

## v10.5

- Tenant-isolated programs, RF telemetry/anomalies, orbital events, asset-CVE alert operations and mixed global/local search.

## v10.4

- Tenant-isolated agent administration and organization ownership across enrollment, certificates, remote config and legacy agent keys.

## v10.3

- Tenant-isolated assets, incidents and audit handlers; corrected v10.2 agent ownership migration to `agent_identities`.

## v10.2

- Added mandatory organization ownership and tenant-first indexes for agents, assets, incidents and audit, plus reusable tenant-scope query primitives.

## v10.1

- Tenant-scoped user administration with per-organization list/create/role/status/delete controls and last-admin protection.

## v10.0

- Organizations / multi-tenancy foundation with default-org migration, user membership, JWT organization context and admin organization API.

## v9.5

- Added maneuver/conjunction context persistence and bounded viewer/operator API endpoints; events remain candidate-only evidence.
- Extended cross-domain correlation with satellite-specific orbital-operations evidence while preserving severity safety gates.
- Fixed automatic evidence persistence to use the database-supported `external` evidence type instead of silently rejected custom types.

## v9.4 cross-domain correlation

- Added an explainable cross-domain correlation summary to automatically generated RF detection incidents.
- Correlation now reports confidence and corroborating domains across RF/detection rules, orbital evidence, agent posture, and recent agent audit activity.
- Recent unacknowledged asset CVE alerts can be attached as explicitly `context_only` evidence and never increase severity by themselves.
- Cross-domain severity remains candidate-only; existing orbital/TLE safety gates and incident severity policy remain authoritative.

## v9.3 pass prediction / v9.2.2 TLE freshness

- Added satellite pass prediction with AOS/LOS, peak elevation, configurable station masks, bounded prediction horizons, and v9.2.2 TLE-quality gating.
- Added TLE epoch persistence and configurable freshness states derived from the orbital element itself rather than ingestion time.
- Suppressed orbital severity escalation for aging, stale, unusable, or unvalidated orbital context and disabled orbital auto-escalation by default.
- Added TLE age/staleness and CelesTrak synchronization metrics plus Prometheus alerts for degraded orbital data quality.
- Exposed TLE quality and orbital-confidence fields in satellite responses and omitted propagated positions for stale/unusable TLEs.
- Refined the README into a concise project landing page with product context, CI/license/release/coverage badges, the existing dashboard screenshot, a development-only quick-start warning, visible contribution guidance, and direct roadmap links.

## v9.1.1 backend runtime image hardening

- Replaced the backend distroless runtime dependency with a scratch image to remove the failing GCR runtime pull from CI.
- Copied only CA certificates, timezone data, the static Go binary, and migrations into the runtime image.
- Kept the backend non-root with numeric UID/GID 65532 and stripped build paths/symbols from the production binary.

## v9.1 secure remote agent management

- Added declarative, versioned SDR agent configuration for capture frequency and interval with strict server/database/agent bounds.
- Added independent Ed25519 signing of remote configuration; agents verify signature, checksum, and bound `agent_id` before applying changes.
- Added mTLS-only agent config polling and acknowledgements, immutable version history, acknowledgement state, and audit events.
- Added rollback by creating a new signed version from historical settings rather than mutating history.
- Added runtime application without process restart and no remote shell/command execution surface.
- Added Connectivity administration controls, signing-key generation tooling, OpenAPI coverage, and remote configuration documentation.

## v9.0

- Added one-time SDR agent enrollment with hashed, expiring tokens.
- Added agent-generated ECDSA keys, CSR signing, client certificates, and mTLS authentication.
- Added automatic certificate rotation, quarantine, irreversible revocation, certificate fingerprints, versions, and last-seen posture.
- Added mTLS identity management to Connectivity while retaining legacy API keys only for migration.
- Added migration `0021_agent_identity_mtls.sql` and SDR Agent CA generation tooling.

## v8.5 pre-v1 stabilization

- Reduced dashboard backend load by polling only the active workspace and opening satellite/incident SSE streams only while those views are active.
- Added composite PostgreSQL indexes for RF history/anomalies, audit actions, satellite freshness, password-reset expiry, and WebAuthn credential usage.
- Added backup verification, guarded restore tooling, and a full-stack backup/restore drill against an isolated temporary database.
- Expanded authorized load testing with duration mode, authenticated incident/search/satellite cases, and optional result capture.
- Added a sensitive-endpoint audit inventory and dedicated backup/restore operations guide.

## v8.4 WebAuthn, recovery, tracing, and performance

- Added password-plus-WebAuthn authentication for cross-platform USB/NFC security keys with user verification and Redis-backed one-time ceremonies.
- Added password re-authentication for security-key enrollment/removal, atomic authorization-version changes, and session revocation after key-management changes.
- Added opt-in TLS SMTP password recovery with hashed 15-minute single-use tokens, Google Authenticator verification when enabled, and administrator-assisted recovery for WebAuthn-only accounts.
- Added optional OTLP/HTTP OpenTelemetry tracing with W3C trace propagation and response trace IDs.
- Added authenticated global search across satellites, assets, CVEs, and incidents plus targeted search indexes.
- Bounded satellite position calculation with a fixed worker pool and added an authorization-gated load-test helper.
- Added database integration coverage for WebAuthn handles, single-use recovery tokens, and global search.

## v8.3.1 smoke-test compatibility

- Updated the authentication smoke test to exercise the mandatory bootstrap password-change flow before accessing protected session endpoints.
- Kept the production `must_change_password` restriction intact and verified that the bootstrap token is rejected from non-password-management endpoints.
- Continued the Google Authenticator E2E flow with the rotated smoke-test password instead of the insecure bootstrap credential.

## v8.3 identity and authorization hardening

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

## v8.1 pre-v1 security baseline

- Added CycloneDX SBOM generation for all Go modules and the frontend in CI and tagged releases.
- Added dependency review, critical SDR coverage gating, and active SDR fuzzing to CI.
- Added an authentication E2E smoke flow covering login, `/auth/me`, logout revocation, and relogin.
- Added JWT issuer/audience/subject validation, configurable short access-token TTL, and previous-key overlap for controlled signing-key rotation.
- Added user-wide session revocation after password, TOTP, and role changes and fail-closed production Redis startup.
- Added a formal STRIDE threat model, compliance readiness checklist, and software supply-chain guide.
- Hardened forensic replay after fuzzing found non-finite IQ propagation and unbounded capture filenames; forensic files now use restrictive permissions.

## v8 detection engineering

- Added typed RF detection rules with admin-only mutation and viewer-readable configuration.
- Added persisted rule matches, per-rule/per-agent cooldown suppression, and bounded active-rule evaluation.
- Added deterministic correlation that collapses multiple rule matches for one anomaly into one incident using the highest configured severity.
- Added detection-rule and anomaly evidence to correlated incidents and system audit events for automatic incident creation.
- Added Detection Engineering controls to the Signal dashboard and synchronized EN/ES/FR/DE translations and OpenAPI.

## v7.1 hardening

- Enforced incident lifecycle transitions in Go and PostgreSQL.
- Added idempotent incident creation and immutable typed evidence references.
- Added readiness checks and structured request correlation logs without request payloads.
- Reduced default/max pagination to 25/100.
- Replaced SDR TLS verification bypass with optional trusted CA configuration.
- Hardened database backups to avoid credential echoing and use restrictive file permissions.
- Added race detector and pinned gosec security gates to CI.

## v7 and earlier — initial platform baseline

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
## Block 7 - Redis TLS and authentication

- Added authenticated `rediss://` support to the backend and CVE worker with TLS 1.2 minimum, system CA verification, optional private CA loading and server-name override.
- Production now rejects plaintext or unauthenticated Redis URLs.
- Redis credentials are redacted from logs and connection diagnostics.
- Helm supports optional Redis CA mounting for both workloads and the CVE worker now consumes `REDIS_URL` consistently.
- AWS ElastiCache uses required transit encryption and validates AUTH-token shape while remaining opt-in.
- Added Redis TLS/auth contracts and regression tests.


## Block 8 - Kubernetes security hardening

- Completed Pod Security Standards restricted-compatible security contexts for backend, frontend, CVE sync and the backend secret-preparation init container.
- Disabled service-link injection and explicit host network, PID and IPC sharing for application Pods.
- Added namespace Pod Security Admission enforcement material for production and warn/audit labels for the local kind workflow.
- Reworked NetworkPolicy into default-deny ingress and egress isolation with explicit DNS, frontend-to-backend, PostgreSQL, Redis, HTTPS, ingress-controller, monitoring and SDR-agent paths.
- Production and AWS overlays now enable NetworkPolicy by default; the AWS example scopes ALB and managed-datastore traffic to the VPC CIDR.
- Added Helm fail-closed validation for production NetworkPolicy, datastore CIDRs and direct SDR-agent ingress CIDRs.
- Added Kubernetes security-hardening contracts and regression tests.

## Block 10 - AWS Secrets Manager and External Secrets

- Added External Secrets Operator installation through Terraform with a pinned Helm chart, dedicated namespace and EKS Pod Identity.
- Restricted the operator IAM policy to read-only access to the Lyrahcle application secret and KMS decryption.
- Added Helm `SecretStore` and `ExternalSecret` resources that preserve the existing `lyrahcle-app-secrets` application contract.
- Kept Terraform free of secret values; application secret material remains populated out-of-band in AWS Secrets Manager.
- Added fail-closed Helm validation, contracts and operational documentation for external secret synchronization.
- Updated the root README dashboard screenshot to the current Mission Security Operations interface.
