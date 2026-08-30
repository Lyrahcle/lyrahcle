# TLE data quality

Orbital security conclusions are only as reliable as their orbital elements. Lyrahcle treats TLE freshness as an explicit security input instead of assuming that the most recently downloaded record is current.

The backend parses the epoch embedded in TLE line 1 and stores it in `satellites.tle_epoch`. Database `updated_at` remains an ingestion timestamp and is not used as proof of orbital freshness.

## States

- `fresh`: within `TLE_CORRELATION_MAX_AGE`; high-confidence input.
- `aging`: older than the correlation threshold but not yet stale; degraded decision-support context only.
- `stale`: older than `TLE_STALE_MAX_AGE`; propagation is suppressed.
- `unusable`: no valid epoch or an implausible future epoch; propagation is suppressed.

## Safe default

`ORBITAL_AUTO_ESCALATION_ENABLED=false` keeps all orbital severity findings in candidate mode. This is the recommended state until a deployment has validated Doppler and visibility behavior against representative physical RF captures from its own receivers, antennas, clocks, and station geometry.

When automatic escalation is explicitly enabled, only `fresh` TLEs pass the data-quality gate.

## Operational response

When a stale-data alert fires:

1. verify CelesTrak synchronization health;
2. inspect the TLE epoch rather than only the last sync timestamp;
3. keep orbital escalation disabled if freshness cannot be restored;
4. treat existing orbital evidence as degraded until a valid TLE is available.
