# Development Plan

Lyrahcle is moving from a collection of space-security capabilities toward a coherent Space Security Operations platform. The detailed shipped/partial/next/later backlog is maintained in `docs/improvement-roadmap.md`; this file only explains execution order and must not duplicate that backlog.

## Current foundation (shipped)

- Orbital tracking with live SSE updates, pass prediction (AOS/LOS), and maneuver/conjunction context
- RF ingestion, statistical anomaly detection, typed detection rules, and orbital-aware Doppler/TLE-freshness gating
- SDR agent mTLS identity, automatic certificate rotation, quarantine/revocation, signed remote configuration, and staged fleet rollouts
- TM/TC (PUS/CSP) decoding
- Space CVE intelligence, asset alerts, and satellite security profiles with explainable mission risk
- JWT, TOTP, WebAuthn, password recovery, RBAC with custom roles/capabilities, multi-tenant organizations with PostgreSQL row-level security, and full audit logging
- Explainable cross-domain incident correlation and the interactive correlation constellation graph
- Cloud-neutral Kubernetes/Helm deployment plus AWS Terraform/OpenTofu foundation (VPC/EKS/ECR/RDS/KMS/Secrets Manager)
- EN/ES/FR/DE dashboard localization, shared UX primitives, and core accessibility work

Full detail and version history for every item above lives in `docs/improvement-roadmap.md` and `CHANGELOG.md`.

## Current execution order

Ordered by what blocks v1.0, per `docs/improvement-roadmap.md` → "Current pre-v1 priorities":

1. Validate orbital-aware Doppler/visibility correlation with representative physical SDR captures before enabling automatic orbital escalation (`ORBITAL_AUTO_ESCALATION_ENABLED=false` until then). The RTL-SDR evidence harness is delivered; the physical capture campaign remains open.
2. Reach and hold >80% coverage on security-critical code paths, including a broader security end-to-end matrix beyond the current HTTP tenant-boundary/IDOR/RBAC gate.
3. Exercise multi-replica/AWS resilience and collect DR/RPO/RTO evidence before cutover.
4. Complete the agent hardware-capability physical lab-validation gate (RTL-SDR/HackRF) — `compatible` currently reflects self-reported software capability only.
5. Wire the frontend warning gate into CI and widen it beyond `act(...)` to `console.error`/`console.warn`/unhandled promises.
6. Stand up a small Playwright suite (5-8 journeys) covering the core detection-to-incident-closure operator workflow; no browser-driven journey coverage exists today.
7. Instrument per-route query-count/rows-examined/cache-hit-miss metrics and document performance budgets for the critical read paths (latency histograms are already shipped).
8. Move the remaining version-suffixed `cmd/server/*.go` files into domain packages under `internal/`, one domain at a time.
9. Finish remaining operator-facing user guides and the outstanding WCAG 2.2 AA accessibility review.

## v1.0 gate

v1.0 requires green CI, production-safe defaults, tested recovery procedures, real SDR validation, documented trust boundaries, a stable operator workflow for detection through investigation, and the execution-order items above closed.
