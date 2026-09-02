# Resilience and disaster recovery

Lyrahcle treats availability, graceful degradation and recoverability as separate controls. This document defines the current local resilience drills and the design targets for AWS. The AWS values are design objectives and are not production-certified until a real production-like exercise records evidence against them.

## Targets

The machine-readable targets live in `deploy/resilience/targets.json`.

| Scenario | Target |
|---|---:|
| Backend pod replacement RTO | 60 s |
| Frontend pod replacement RTO | 60 s |
| CVE sync pod replacement RTO | 90 s |
| Worker drain application RTO | 90 s |
| PostgreSQL outage detection | 30 s |
| PostgreSQL recovery RTO after connectivity restoration | 90 s |
| Redis outage detection | 30 s |
| Redis recovery RTO after connectivity restoration | 60 s |
| AWS application RTO design objective | 15 min |
| AWS PostgreSQL RPO design objective | 5 min |
| AWS PostgreSQL restore RTO design objective | 60 min |
| AWS Redis failover RTO design objective | 15 min |

RTO measures the time required to restore an expected service state. RPO measures acceptable durable data loss. Local connectivity drills do not destroy persistent storage, so they do not claim to validate a production storage-loss RPO.

## Local kind chaos drill

Run `./scripts/k8s-resilience-test.sh` only after `./scripts/k8s-local-up.sh` has created the local three-node kind environment.

The drill performs these controlled faults:

1. Deletes one backend pod and measures full replica recovery.
2. Deletes one frontend pod and measures full replica recovery.
3. Deletes the CVE sync pod and measures recovery.
4. Validates that backend and frontend replicas are distributed across at least two nodes and that each PDB permits one voluntary disruption.
5. Drains one worker only for Lyrahcle application pods and measures recovery while the PDBs are enforced.
6. Uses service-selector isolation for Redis, terminates existing Redis client connections without deleting data, verifies backend readiness fails within the target without container restarts, restores the Service and measures readiness recovery.
7. Uses service-selector isolation for PostgreSQL, terminates existing sessions for the local application database without deleting data, verifies backend readiness fails within the target while liveness does not force container restarts, restores the Service and measures readiness recovery.

The cleanup trap restores both Service selectors and uncordons a drained node even if the drill fails. The report is written to `/tmp/lyrahcle-k8s-resilience-report.json`.

The Redis drill complements the Go integration tests. Kubernetes readiness intentionally fails when Redis is unavailable because `/readyz` treats Redis as a required dependency. Package-level tests remain responsible for proving bounded local rate-limit fallback and fail-closed behavior for security-critical Redis checks before or outside Kubernetes traffic removal.

## Local restore drill

Run `./scripts/dr-local-restore-test.sh` with the Docker Compose PostgreSQL service available.

The drill is deliberately isolated from the primary local database. It:

1. Uses `lyrahcle_test` as the source.
2. Produces and verifies a compressed PostgreSQL dump.
3. Creates a uniquely named temporary database.
4. Restores the dump into that temporary database.
5. Compares every user table and exact row count between source and restored databases.
6. Compares the `schema_migrations` count.
7. Removes the temporary database on success or failure.

The report is written to `/tmp/lyrahcle-dr-restore-report.json`.

This proves recoverability at the backup boundary. Production RPO still depends on the real backup and point-in-time-recovery cadence.

## Consolidated CYSAT DR evidence

After both local drills pass, consolidate their machine-readable reports with:

```bash
python3 scripts/collect-dr-evidence.py
```

The command reads `/tmp/lyrahcle-k8s-resilience-report.json` and `/tmp/lyrahcle-dr-restore-report.json`, rejects missing or non-PASS evidence, rejects any kind RTO measurement above its declared target, and writes `/tmp/lyrahcle-dr-evidence.json`. The collector itself is regression-tested in CI.

A PASS evidence file supports the following bounded claim for a CYSAT demo: application/dependency recovery targets were measured in the local three-node kind environment, and the PostgreSQL restore drill observed zero lost rows at the verified backup boundary while checking table counts and migration state. It does **not** certify the AWS design objectives or prove a five-minute production RPO; those remain pending until an isolated RDS/PITR and ElastiCache production-like exercise is performed.

The evidence JSON records each observed RTO beside its target, worst target utilisation, restore duration, integrity counts, source-report timestamps, RPO scope, and the explicit certification limit. Keep the JSON with demo/release evidence when a drill is used as a resilience proof point.

## Kubernetes production hardening

Production requires backend and frontend PDBs with at least one available replica. It also requires strict topology spreading across hostname and zone with at least two eligible domains. The Helm validation fails closed if these controls are weakened.

Staging and local kind keep `ScheduleAnyway` because a two-worker disposable cluster must still be able to recover after one worker is drained. The test still requires the initial healthy state to have replicas on at least two nodes.

## AWS DR foundation

The current Terraform already provides the following foundations:

- RDS PostgreSQL is Multi-AZ in production.
- RDS production backups retain 14 days.
- RDS production deletion protection is enabled and a final snapshot is required.
- ElastiCache has automatic failover and Multi-AZ enabled.
- ElastiCache production snapshots retain seven days.

These controls reduce risk but do not by themselves prove the AWS RTO/RPO objectives. A later production-like DR exercise must restore RDS to an isolated target, validate application integrity, record elapsed time, exercise ElastiCache failover and retain the evidence. Destructive AWS chaos is intentionally outside the local script and is never run by the normal CI gate.

## Scope limits

The local kind drill does not simulate an AWS Availability Zone loss, RDS instance replacement, point-in-time restore, KMS outage, Secrets Manager outage or regional disaster. It does not run `terraform apply`, modify AWS resources or destroy namespaces. The local PostgreSQL/Redis faults preserve storage by changing Service selectors and terminating existing client connections so connection pools cannot mask the simulated outage.
