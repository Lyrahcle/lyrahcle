# Community Reception Import

Lyrahcle can import third-party ground reception from the SatNOGS Network and hold it as
**context-only evidence**. This gives a mission observability signal without owning a station,
and it is the main way to exercise the orbital model before physical SDR validation is closed.

## What the data is and is not

- It **is** proof that a downlink was observable by someone else at a known time and place.
- It is **not** proof of onboard health, of control, of vulnerability exposure, or, in its
  absence, of an outage.

The schema enforces this: `evidence_kind` is fixed to `external_community`, `applicability`
to `reception_context_only`, and `confidence` is capped at 60 by a database constraint because
the observation was neither made nor calibrated by a Lyrahcle station.

## Endpoints

```
POST /api/v1/satellites/{satelliteID}/external-observations/import   # orbital.manage
GET  /api/v1/satellites/{satelliteID}/external-observations          # orbital.read
```

The import body is optional: `window_days` (default 30, max 90) and `limit` (max 200).
Records are upserted on `(organization_id, source, external_id)`, so re-running an import is
idempotent. Every import writes an `external_reception.imported` audit entry.

Neither community-reception endpoint currently has a frontend panel. Their stable,
tenant-scoped response contracts and the geometry verdicts are intended to feed the future
SPARTA coverage workspace; frontend implementation is deliberately deferred to that block.

## Geometry consistency check

For each record that carries station coordinates, Lyrahcle samples the observation window
against the stored TLE and reports:

- `max_elevation_deg` and `visible` from the propagated geometry,
- `range_rate_mps` at the peak and the resulting `expected_doppler_hz` when the transmitter
  downlink frequency is known,
- a `consistency` verdict.

| Verdict | Meaning |
|---|---|
| `consistent` | A vetted reception coincides with a modelled visibility window. |
| `inconsistent` | A vetted reception falls outside the modelled window — usually stale orbital elements or a mismatched satellite. |
| `unknown` | No vetted reception, no coordinates, or the model could not be evaluated. |

`inconsistent` is a data-quality finding about Lyrahcle's own orbital model, not an incident.
Treat it as a prompt to refresh the TLE before trusting Doppler-based correlation.

## Ingestion safeguards

- Only `https://db.satnogs.org` and `https://network.satnogs.org` are reachable, with bounded
  redirects and a response size limit.
- Future-dated, zero-length, inverted, duplicated and older-than-90-day observations are dropped.
- Free-text fields are trimmed, coordinates and frequencies are range-checked, and artefact
  links are kept only when they are HTTPS.
- Rows are tenant-scoped and protected by PostgreSQL row-level security like the rest of the
  operational tables.
