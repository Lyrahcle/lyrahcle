# Orbital-aware security correlation

Lyrahcle v9.2 adds physical orbital context to RF incidents without replacing the existing deterministic RF detection rules. v9.2.2 adds a TLE data-quality gate so orbital calculations cannot silently raise incident severity from stale orbital elements.

## Trust model

Ground-station coordinates are configured by an administrator on the enrolled SDR identity. The agent does not self-assert its physical location.

The target satellite is distributed through the existing Ed25519-signed remote configuration. The SDR agent verifies the signature, checksum, and bound agent identity before applying the target and then includes its `satellite_id` in RF readings.

If a reading has no satellite target or the station has no configured coordinates, Lyrahcle keeps normal RF detection behavior and omits orbital correlation.

## TLE data quality

Lyrahcle parses the epoch encoded in TLE line 1 and stores it independently from the database update timestamp. Re-fetching an old TLE therefore does not make it appear fresh.

The default policy is:

| Age | Status | Orbital context | Automatic severity |
|---|---|---|---|
| `<=24h` | `fresh` | evaluated | data-quality eligible |
| `24h-72h` | `aging` | evaluated as degraded context | suppressed |
| `>72h` | `stale` | not propagated | suppressed |
| missing/invalid/far-future epoch | `unusable` | not propagated | suppressed |

Thresholds are configured with `TLE_CORRELATION_MAX_AGE` and `TLE_STALE_MAX_AGE`.

`ORBITAL_AUTO_ESCALATION_ENABLED` defaults to `false`. Until representative physical RF validation is complete, orbital findings are decision-support evidence and candidate severity only. Enabling the flag allows a `fresh` TLE to influence severity; aging, stale, and unusable TLEs remain blocked.

## Physical checks

For satellite-tagged anomalies with usable TLE data Lyrahcle derives:

- local elevation and horizon visibility;
- slant range;
- one-second range rate;
- expected Doppler shift at the configured RF frequency.

The correlation records a candidate `critical` severity when the claimed satellite is below the station horizon. Frequency anomalies record a candidate of at least `high` when the residual against the Doppler-aware baseline exceeds the larger of 2.5 kHz or three baseline standard deviations.

The incident evidence records whether candidate severity was applied or suppressed, the TLE epoch and age, freshness state, confidence, and suppression reason.

## Monitoring

Prometheus exports:

- `lyrahcle_tle_age_seconds`;
- `lyrahcle_tle_stale`;
- `lyrahcle_tle_correlation_eligible`;
- `lyrahcle_celestrak_sync_failures_total`;
- `lyrahcle_orbital_correlation_suppressed_total`.

Bundled alerts warn when TLEs are stale/unusable, when CelesTrak synchronization fails, and when TLE quality suppresses orbital correlation.

## Threat intelligence

Recent high/critical space CVEs are attached as contextual evidence only. They do not increase incident severity because Lyrahcle does not yet have a complete component-to-satellite asset graph.

## Operator workflow

1. Enroll the SDR agent with mTLS.
2. In Connectivity, configure the trusted station latitude, longitude, and altitude.
3. In Signed Remote Configuration, choose a tracked satellite along with frequency and capture interval.
4. Confirm the TLE quality state before interpreting orbital findings.
5. Wait for the agent acknowledgement.
6. Review orbital evidence as decision support until the deployment has completed representative RF validation.

## Security properties

- Station coordinates are admin-controlled.
- Satellite targeting is signed independently of mTLS.
- Unknown satellite IDs are rejected by the backend.
- TLE epoch freshness is derived from the orbital element itself, not ingestion time.
- Aging/stale/unusable TLEs cannot raise severity.
- Orbital automatic escalation is disabled by default.
- Generic CVEs cannot escalate severity.
