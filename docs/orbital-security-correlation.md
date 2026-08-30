# Orbital-aware security correlation

Lyrahcle v9.2 adds physical orbital context to RF incidents without replacing the existing deterministic RF detection rules.

## Trust model

Ground-station coordinates are configured by an administrator on the enrolled SDR identity. The agent does not self-assert its physical location.

The target satellite is distributed through the existing Ed25519-signed remote configuration. The SDR agent verifies the signature, checksum, and bound agent identity before applying the target and then includes its `satellite_id` in RF readings.

If a reading has no satellite target or the station has no configured coordinates, Lyrahcle keeps normal RF detection behavior and omits orbital escalation.

## Physical checks

For satellite-tagged anomalies Lyrahcle evaluates the TLE at the capture timestamp and derives:

- local elevation and horizon visibility;
- slant range;
- one-second range rate;
- expected Doppler shift at the configured RF frequency.

A correlated incident receives a `critical` severity floor when the claimed satellite is below the station horizon. Frequency anomalies receive at least `high` when the residual against the Doppler-aware baseline exceeds the larger of 2.5 kHz or three baseline standard deviations.

## Threat intelligence

Recent high/critical space CVEs are attached as contextual evidence only. They do not increase incident severity because v9.2 does not yet have a complete component-to-satellite asset graph.

## Operator workflow

1. Enroll the SDR agent with mTLS.
2. In Connectivity, configure the trusted station latitude, longitude, and altitude.
3. In Signed Remote Configuration, choose a tracked satellite along with frequency and capture interval.
4. Wait for the agent acknowledgement.
5. RF anomalies for that configuration carry the satellite identity into the incident timeline and evidence.

## Security properties

- Station coordinates are admin-controlled.
- Satellite targeting is signed independently of mTLS.
- Unknown satellite IDs are rejected by the backend.
- Orbital correlation is skipped instead of guessed when required context is missing.
- Generic CVEs cannot escalate severity.
