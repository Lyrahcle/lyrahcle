# Development Plan

Lyrahcle is moving from a collection of space-security capabilities toward a coherent Space Security Operations platform.

## Current foundation

- Orbital tracking with live SSE updates
- RF ingestion and statistical anomaly detection
- SDR agent identity and device self-report
- TM/TC decoding
- Space CVE intelligence and asset alerts
- JWT, TOTP, RBAC, audit logging, Redis-backed revocation and rate limiting
- EN/ES/FR/DE dashboard localization

## Next priorities

### 1. Incidents and correlation

Create a first-class incident model that correlates RF anomalies, asset alerts, orbital events, agents, and audit evidence into a single timeline.

### 2. RF detection engineering

Add reusable detection rules, per-satellite or per-link baselines, replay/spoofing indicators, and stronger signal fingerprints.

### 3. Agent identity hardening

Move from long-lived API keys toward enrollment, short-lived credentials or certificates, rotation, revocation, and eventually mTLS.

### 4. Orbital security context

Add pass prediction, expected Doppler windows, visibility checks, maneuver/anomaly signals, and conjunction context.

### 5. Asset graph and exposure

Model relationships between programs, satellites, ground stations, radios, firmware, agents, users, vulnerabilities, and incidents. Use those relationships to prioritize attack paths and exposure.

### 6. Production hardening

Complete real SDR hardware validation, high-availability deployment guidance, stronger observability, signed artifacts, SBOM/provenance, backup/restore drills, and operational runbooks.

## v1.0 gate

v1.0 should require green CI, production-safe defaults, tested recovery procedures, real SDR validation, documented trust boundaries, and a stable operator workflow for detection through investigation.
