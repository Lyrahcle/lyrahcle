# Orbital operations context (v9.5)

Lyrahcle v9.5 adds persistent maneuver and conjunction context for satellites. The feature is intentionally conservative: it stores context supplied by an operator or trusted integration and uses it as explainable candidate evidence. It does **not** claim automatic maneuver detection or autonomous conjunction screening.

## API

- `GET /api/v1/satellites/{satelliteID}/orbital-events` — viewer or higher.
- `POST /api/v1/satellites/{satelliteID}/orbital-events` — operator or higher.

Each event records `event_type` (`maneuver` or `conjunction`), event time, source, confidence from `0` to `1`, severity, arbitrary structured metadata, creator, and creation time. Listing defaults to a seven-day lookback and is bounded to protect the API.

## Correlation behavior

When an RF detection incident references a satellite, v9.4 cross-domain correlation now reads recent orbital operations events for that same satellite. High-confidence maneuver/conjunction context can increase the **candidate** confidence/severity and is attached to the incident as external evidence. `severity_applied` remains `false`.

This preserves the existing safety model: orbital context is useful to operators without bypassing TLE freshness gating or the physical-SDR validation gate for automatic orbital escalation.

## Trust and ingestion

The backend marks events with `candidate_only=true` and `ingestion_mode=operator_or_trusted_integration`. Integrations should preserve provider identifiers and assessment details in `metadata` (for example conjunction object identifiers, miss distance, probability of collision, or maneuver delta-v) so the evidence remains reviewable.

## Remaining work

Automatic maneuver inference requires historical orbital-element comparison and validation thresholds. Automated conjunction screening requires a trusted conjunction/catalog source or an independently validated screening pipeline. Those capabilities remain separate follow-up work rather than being implied by this persistence/API layer.
