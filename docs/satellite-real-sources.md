# Real satellite sources

Lyrahcle enriches a tracked satellite by its NORAD catalog identifier using public SatNOGS services. The satellite security profile requests `/api/v1/satellites/{satelliteID}/external-sources`; the backend normalizes the upstream records and caches the result for 15 minutes. After expiry, the next view refreshes the data automatically.

## Sources

- **SatNOGS DB satellites** supplies the SatNOGS identifier, name aliases, catalog status, countries, image and website metadata.
- **SatNOGS DB transmitters** supplies published uplink/downlink ranges, operating mode, active status and transmitter identity.
- **SatNOGS Network observations** supplies recent public observation windows, station identity, status and transmitter identity.

The integration uses the fixed HTTPS origins `db.satnogs.org` and `network.satnogs.org`, bounded response bodies, a ten-second timeout, a narrow redirect allowlist, encoded query parameters and sanitized source errors. No SatNOGS credential is required for these public endpoints.

## Freshness and quality

Each source reports availability, fetch time, record count and freshness. The latest reception excludes observations that are still marked `future` and excludes timestamps after the current server time.

- `fresh`: completed reception within 24 hours;
- `aging`: completed reception within seven days;
- `stale`: last completed reception is older than seven days;
- `unknown`: the source returned no completed reception;
- `unavailable`: the upstream source could not be safely read.

Overall quality is `good` only when all three sources are available and reception is fresh. Partial availability or an unknown/aging reception is `degraded`; old reception is `stale`; complete source failure is `unavailable`.

## Evidence boundary

SatNOGS observations are external community evidence of reception at a ground station. They do not prove that the spacecraft is healthy, that its payload is functioning correctly, that the observer controls it, or that a CVE applies to its hardware or software. Lyrahcle returns `evidence_kind=external_community` and `applicability=radio_activity_context_only` to keep this boundary machine-readable.
