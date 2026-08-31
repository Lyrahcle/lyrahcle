# Background job reliability

Lyrahcle v1 treats background processing as **at-least-once delivery with idempotent durable effects**. A job may execute more than once after a crash or lease expiry; it must not create duplicate tenant data or duplicate audit side effects.

## CVE alert events

`cve-sync` publishes CVE IDs to the Redis Stream `lyrahcle:cve-events:v2`. Backend alert workers consume the stream through the `asset-alert-workers` consumer group.

A stream message is acknowledged only after `ProcessCVE` completes. If a backend process dies after receiving an event but before ACK, the message remains pending and another consumer reclaims it after the idle lease. Failed processing is deliberately left pending for retry.

During rolling upgrades, backend workers also migrate entries from the legacy Redis list `lyrahcle:cve-events`. The RPOP + XADD migration is a single Redis Lua operation, so a process crash cannot occur between removal from the list and insertion into the stream.

Duplicate stream delivery is expected and safe: `asset_cve_alerts` uses the durable `(asset_id, cve_id)` uniqueness constraint and audit creation happens only when the alert row is newly inserted.

## RF detection and incident correlation

`rf_detection_matches` is unique by `(rule_id, anomaly_id)`. A retry of the same anomaly first recovers its existing match before evaluating cooldown; this prevents a crash after match insertion from making the retry appear suppressed.

Automatic incidents continue to deduplicate on the source anomaly. `CreateWithResult` reports whether the incident was newly created so system audit creation is emitted only once. Incident evidence already uses `(incident_id, evidence_type, evidence_ref)` idempotency.

HTTP incident idempotency keys are scoped by `organization_id`; the same key can be used independently by two tenants without collision or cross-tenant lookup.

## Singleton scheduled jobs

Every backend replica contains the scheduler, but jobs with global side effects acquire a Redis lease before execution. The lease uses `SET NX` with an expiry and an owner token. Unlock uses a compare-and-delete Lua script so an expired owner cannot delete a lease acquired later by another replica.

Currently protected jobs include:

- `tle-sync`
- `signing-key-rotation-check`
- `recon-sweep` placeholder

Production already requires Redis. In development without Redis the locker allows local single-process execution; this mode is not a high-availability guarantee.

## Operational metrics

- `lyrahcle_background_job_events_total{job,result}`
- `lyrahcle_scheduled_job_leadership_total{job,result}`

Useful results include `processed`, `failed`, `reclaimed`, `ack_failed`, `legacy_migrated`, `acquired`, `skipped`, `error`, and `release_error`.
