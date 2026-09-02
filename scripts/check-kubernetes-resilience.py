#!/usr/bin/env python3
import json
import sys
from pathlib import Path

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
errors = []


def read(path):
    target = root / path
    if not target.exists():
        errors.append(f"missing required path: {path}")
        return ""
    return target.read_text(encoding="utf-8")


def require(text, needle, label):
    if needle not in text:
        errors.append(f"{label}: missing {needle!r}")


def forbid(text, needle, label):
    if needle in text:
        errors.append(f"{label}: forbidden {needle!r}")


target_path = root / "deploy/resilience/targets.json"
if not target_path.exists():
    errors.append("missing resilience targets")
    targets = {}
else:
    try:
        targets = json.loads(target_path.read_text(encoding="utf-8"))
    except Exception as exc:
        errors.append(f"invalid resilience targets: {exc}")
        targets = {}

if targets:
    if targets.get("status") != "design-targets-not-production-certified":
        errors.append("resilience targets must remain explicitly uncertified for production")
    local = targets.get("local_kind", {})
    required_local = {
        "backend_pod_rto_seconds": (1, 120),
        "frontend_pod_rto_seconds": (1, 120),
        "cve_sync_pod_rto_seconds": (1, 180),
        "node_drain_application_rto_seconds": (1, 180),
        "postgres_outage_detection_seconds": (1, 60),
        "postgres_recovery_rto_seconds": (1, 180),
        "redis_outage_detection_seconds": (1, 60),
        "redis_recovery_rto_seconds": (1, 120),
    }
    for key, bounds in required_local.items():
        value = local.get(key)
        if not isinstance(value, (int, float)) or not bounds[0] <= value <= bounds[1]:
            errors.append(f"invalid local resilience target {key}: {value!r}")
    aws = targets.get("aws_design", {})
    for key in ("application_rto_minutes", "postgres_rpo_minutes", "postgres_restore_rto_minutes", "redis_failover_rto_minutes"):
        value = aws.get(key)
        if not isinstance(value, (int, float)) or value <= 0:
            errors.append(f"invalid AWS design target {key}: {value!r}")

values = read("deploy/helm/lyrahcle/values.yaml")
production = read("deploy/helm/lyrahcle/values-production.yaml")
aws_values = read("deploy/helm/lyrahcle/values-aws.example.yaml")
backend = read("deploy/helm/lyrahcle/templates/backend-deployment.yaml")
frontend = read("deploy/helm/lyrahcle/templates/frontend-deployment.yaml")
cve = read("deploy/helm/lyrahcle/templates/cve-sync-deployment.yaml")
validate = read("deploy/helm/lyrahcle/templates/validate.yaml")

for component, text in (("backend", backend), ("frontend", frontend)):
    require(text, f".Values.{component}.topologySpread.zoneWhenUnsatisfiable", component)
    require(text, f".Values.{component}.topologySpread.hostnameWhenUnsatisfiable", component)
    require(text, f".Values.{component}.topologySpread.zoneMinDomains", component)
    require(text, f".Values.{component}.topologySpread.hostnameMinDomains", component)
    require(text, "topology.kubernetes.io/zone", component)
    require(text, "kubernetes.io/hostname", component)

require(cve, "replicas: {{ .Values.cveSync.replicaCount }}", "CVE sync")
for needle in (
    "zoneWhenUnsatisfiable: ScheduleAnyway",
    "hostnameWhenUnsatisfiable: ScheduleAnyway",
    "zoneMinDomains: 0",
    "hostnameMinDomains: 0",
):
    require(values, needle, "default topology spread")
for needle in (
    "zoneWhenUnsatisfiable: DoNotSchedule",
    "hostnameWhenUnsatisfiable: DoNotSchedule",
    "zoneMinDomains: 2",
    "hostnameMinDomains: 2",
):
    require(production, needle, "production topology spread")
for needle in (
    "zoneWhenUnsatisfiable: DoNotSchedule",
    "hostnameWhenUnsatisfiable: DoNotSchedule",
    "zoneMinDomains: 2",
    "hostnameMinDomains: 2",
):
    if aws_values.count(needle) < 2:
        errors.append(f"AWS production topology spread: expected backend and frontend {needle!r}")
for needle in (
    "production requires backend.pdb.enabled=true",
    "production requires frontend.pdb.enabled=true",
    "production requires strict backend zone topology spread",
    "production requires strict backend hostname topology spread",
    "production requires strict frontend zone topology spread",
    "production requires strict frontend hostname topology spread",
    "production requires backend topology spread across at least two zones",
    "production requires frontend topology spread across at least two zones",
):
    require(validate, needle, "production resilience validation")

chaos = read("scripts/k8s-resilience-test.sh")
for needle in (
    "trap restore_faults EXIT",
    "kubectl --context \"$CONTEXT\" drain",
    "--pod-selector=app.kubernetes.io/instance=lyrahcle",
    "resilience-blocked-redis",
    "resilience-blocked-postgres",
    "redis-cli CLIENT KILL TYPE normal",
    "pg_terminate_backend(pid)",
    "redis_outage_detection_seconds",
    "wait_deployment_not_ready lyrahcle-backend",
    "restart_total",
    "/tmp/lyrahcle-k8s-resilience-report.json",
    "No persistent storage was destroyed",
    "terminate existing client connections only",
):
    require(chaos, needle, "Kubernetes resilience drill")
for needle in ("aws rds", "aws elasticache", "terraform apply", "kubectl delete namespace"):
    forbid(chaos.lower(), needle, "Kubernetes resilience drill")

dr = read("scripts/dr-local-restore-test.sh")
for needle in (
    'SOURCE_DATABASE="lyrahcle_test"',
    'TARGET_DATABASE="lyrahcle_dr_',
    "trap cleanup EXIT",
    "pg_dump -U lyrahcle \"$SOURCE_DATABASE\"",
    "verify-backup.sh",
    "snapshot_database \"$SOURCE_DATABASE\"",
    "snapshot_database \"$TARGET_DATABASE\"",
    "cmp -s \"$SOURCE_SNAPSHOT\" \"$TARGET_SNAPSHOT\"",
    "schema_migrations",
    "/tmp/lyrahcle-dr-restore-report.json",
):
    require(dr, needle, "local DR restore drill")
for needle in ("lyrahcle\"", "RESTORE_ACK", "ALLOW_REMOTE_RESTORE"):
    if needle == "lyrahcle\"":
        continue

rds = read("deploy/terraform/aws/rds.tf")
redis = read("deploy/terraform/aws/redis.tf")
for needle in (
    'multi_az                     = var.environment == "production" ? true',
    'backup_retention_period      = var.environment == "production" ? 14 : 7',
    'deletion_protection          = var.environment == "production" ? true',
    'skip_final_snapshot          = var.environment != "production"',
):
    require(rds, needle, "AWS PostgreSQL DR foundation")
for needle in (
    "automatic_failover_enabled = true",
    "multi_az_enabled           = true",
    'snapshot_retention_limit = var.environment == "production" ? 7 : 1',
):
    require(redis, needle, "AWS Redis resilience foundation")

readme = read("docs/README.md")
doc = read("docs/resilience-disaster-recovery.md")
require(readme, "[Resilience and disaster recovery](./resilience-disaster-recovery.md)", "documentation index")
for needle in (
    "Local kind chaos drill",
    "Local restore drill",
    "RTO",
    "RPO",
    "not production-certified",
    "service-selector isolation",
    "existing client connections",
):
    require(doc, needle, "resilience documentation")

if errors:
    print("kubernetes resilience/DR contract: ERROR")
    for error in errors:
        print(error)
    raise SystemExit(1)

print("kubernetes resilience/DR contract: OK")
