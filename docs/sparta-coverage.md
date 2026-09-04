# SPARTA Coverage

Lyrahcle maps its own evidence onto [SPARTA](https://sparta.aerospace.org/), the space-system
adversary framework published by The Aerospace Corporation, so that operators can answer
"which parts of the space attack surface are we actually able to see?" rather than only
"how many alerts fired?".

## Scope of the catalog

`internal/sparta` ships a **manually maintained curated subset** of the published matrix: nine tactics and the
techniques Lyrahcle can produce evidence for. The subset is versioned in `CatalogVersion`
and every API response repeats `catalog_scope_note`.

**Before using these identifiers in a report, procurement document or tender, reconcile them
against the published SPARTA matrix.** The catalog in this repository is constructed and
maintained by hand; it is not a mirror of the published data.

## Mapping rules

Mapping is deterministic and explainable. Each rule declares the evidence kind, the matching
value, the techniques it implies, a bounded confidence and a written rationale. Rules are
published through `GET /api/v1/sparta/matrix` so an analyst can audit the attribution rather
than trusting a black box.

| Evidence kind | Source | Example match |
|---|---|---|
| `rf_anomaly` | detection engine | `power_dbm`, `frequency_hz`, `snr_db`, `occupied_bandwidth_hz` |
| `incident` | incident pipeline | `rf_anomaly`, `agent_posture`, `orbital_event`, `manual` |
| `tmtc_finding` | PUS/CSP/CCSDS decoders | `sequence_repeat`, `unauthenticated_command`, `malformed_frame` |
| `agent_posture` | SDR fleet | `certificate_auth_failed`, `config_signature_rejected`, `offline` |
| `cve_context` | threat intelligence | `ground_software`, `flight_software` |
| `external_reception` | community networks | `community_observation` |

Two invariants hold:

- Confidence never exceeds **75**. Lyrahcle does not claim certainty about adversary intent.
- Raw RF anomalies stay `context_only` until the physical validation gate in
  `docs/rf-real-acquisition.md` is closed. Anomaly magnitude raises confidence, never
  the context-only flag.

## Coverage matrix

`GET /api/v1/sparta/coverage?window_days=30` builds a tenant-scoped matrix from RF anomalies,
incidents and imported community reception in the requested window (capped at 90 days).

- `evidence_count` deduplicates by evidence reference, so the same incident does not inflate coverage.
- `detectable` marks techniques that at least one mapping rule can reach; the gap between
  `detectable_total` and `technique_total` is the honest description of what Lyrahcle cannot see today.
- `context_only` on an observed technique means every piece of evidence behind it was contextual.

Coverage is a visibility measure, not a security score. A technique with zero evidence may
mean the adversary is absent, or that Lyrahcle has no sensor for it.

## Frontend status and next step

Neither SPARTA endpoint currently has a frontend panel. The backend contracts, metadata and
coverage aggregation are ready for a future coverage workspace that combines the catalog,
tenant evidence and visibility gaps. That workspace is the next natural product increment;
it is intentionally outside this backend/documentation change.
