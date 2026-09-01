# Risk scoring

Lyrahcle calculates deterministic operational risk scores from evidence already available to the active organization and role. Scores range from 0 to 100 and are designed to remain explainable rather than predictive.

## Mission score

The mission score combines active critical and high-severity incidents, severe RF anomalies, urgent exposure alerts, agent trust or availability problems, degraded orbital data quality, and backend availability. Each factor has a bounded contribution so one noisy source cannot grow without limit.

The dashboard shows the resulting score as nominal, elevated, or critical. Domain totals preserve where the score came from: security, RF, exposure, ground segment, orbital, and platform.

## Satellite score

The satellite score combines TLE quality, satellite-scoped incidents, recent severe RF anomalies, high-severity orbital events, and conservative exact-match exposure alerts. The same calculation is reused in the satellite security profile and satellite inventory.

Exposure evidence remains contextual. A CVE or alert is not treated as an onboard vulnerability merely because a satellite exists; the current implementation only includes an alert when the asset value exactly matches the satellite name or NORAD identifier.

## Ground station score

The ground station score reuses the shared risk engine for SDR agents with trusted station coordinates. It combines agent trust or availability problems, active critical and high-severity incidents scoped to the station agent, and severe RF anomalies from the last 24 hours.

Ground Station Operations shows the score alongside station availability, hardware context and current orbital visibility. The score only uses evidence associated with that station's agent identifier and does not infer relationships between independent stations.

## Thresholds

- 0–34: Nominal
- 35–69: Elevated
- 70–100: Critical

The score is an operator prioritization aid. It does not replace incident severity, detection-rule decisions, or mission safety procedures.
