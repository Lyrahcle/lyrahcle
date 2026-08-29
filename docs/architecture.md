# Architecture

Lyrahcle is a small service-oriented monorepo built around a Go API, PostgreSQL, Redis, a React frontend, an SDR edge agent, and an independent CVE synchronization worker.

## Runtime topology

```text
Browser
  |
  v
Frontend (React/nginx)
  |
  v
Backend (Go/chi) <---- SDR Agent
  |      |               |
  |      +---- Redis ----+
  |      |
  |      +---- Prometheus ---- Grafana
  |
  +---- PostgreSQL <---- CVE Sync ---- NVD
  |
  +---- Celestrak
```

## Ownership

| Component | Owns |
|---|---|
| Backend | API, authentication, RBAC, Google Authenticator TOTP, session management, orbital propagation, RF anomaly detection, detection-rule evaluation, incident correlation, audit log, asset alerts |
| Frontend | Operator UX and session state |
| SDR agent | RF capture source, device self-report, IQ recording/replay, authenticated uplink |
| CVE sync | Writes `space_cves`, publishes newly synchronized CVEs, exposes worker metrics |
| PostgreSQL | Durable application state and migrations |
| Redis | Distributed rate limiting, JWT revocation, active-session metadata, response caching, CVE event queue |

`apps/cve-sync` is the only writer of `space_cves`. The backend reads that data and correlates it with assets.

## Trust boundaries

Human sessions use short-lived JWT bearer tokens. Redis stores active-session metadata and revocation state; users can revoke individual sessions or all sessions. Roles are `viewer`, `auditor`, `analyst`, `operator`, and `admin`, with explicit permissions rather than relying on frontend visibility. SDR agents should use individual `X-Agent-Key` credentials; the backend resolves the agent identity from the key and does not trust an arbitrary agent identity for device self-report.

The backend applies request-size limits, input validation, security headers, rate limiting, audit events, request correlation IDs, and authorization at route boundaries.

## Live orbital flow

```text
Celestrak -> TLE sync -> PostgreSQL -> SGP4 propagation -> /satellites/stream -> Dashboard
```

Satellite stream updates use Server-Sent Events. The HTTP middleware stack preserves `http.Flusher` so streaming remains functional through metrics instrumentation.

## RF flow

```text
SDR source -> sdr-agent -> authenticated RF ingest -> baseline analysis -> anomaly -> typed detection rules -> correlated incident -> Dashboard
```

Detection rules are data, not executable expressions. Admins select a supported RF metric, z-score operator, threshold, severity, cooldown, and whether a match may create an incident. Match history is persisted, cooldown checks are serialized per rule and agent, and multiple rule matches for one anomaly collapse into one incident.

The current default source is simulated. The agent abstraction is designed so real SDR hardware can replace the mock source without changing the backend contract.

## Threat intelligence flow

```text
NVD -> cve-sync -> PostgreSQL + Redis event -> backend matcher -> asset alert -> operator acknowledgement
```

## Database changes

Backend migrations in `apps/backend/migrations` are ordered, forward-only, and tracked in `schema_migrations`. Tests use the same migration chain rather than maintaining a separate schema.
