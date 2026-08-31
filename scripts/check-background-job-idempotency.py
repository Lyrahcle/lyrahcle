#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
checks = {
    "backend/internal/alerts/worker.go": [
        '"lyrahcle:cve-events:v2"',
        "XGroupCreateMkStream",
        "XReadGroup",
        "XAutoClaim",
        "XAck",
        "XDel",
        "drainLegacyQueue",
        "migrateLegacyEventScript",
        'WithLabelValues("cve-alerts", "reclaimed")',
        "InsertIdempotentInOrganization",
    ],
    "cve-sync/internal/cache/cache.go": [
        'alertEventsStream = "lyrahcle:cve-events:v2"',
        "XAdd",
        '"event_id"',
    ],
    "backend/internal/joblock/redis.go": [
        "SetNX",
        "releaseScript",
        'defaultPrefix = "lyrahcle:job-lock:"',
    ],
    "backend/cmd/server/main.go": [
        "runSingletonScheduledJob",
        '"tle-sync"',
        '"signing-key-rotation-check"',
    ],
    "backend/internal/store/detection_rules.go": [
        "WHERE organization_id = $1 AND rule_id = $2 AND anomaly_id = $3",
    ],
    "backend/internal/store/incidents.go": [
        "CreateWithResult",
        'case "idx_incidents_org_idempotency_key"',
        "WHERE i.organization_id = $1 AND i.idempotency_key = $2",
    ],
    "backend/internal/store/audit.go": [
        "InsertIdempotentInOrganization",
        "ON CONFLICT (organization_id, dedupe_key)",
    ],
    "backend/migrations/0037_background_job_idempotency.sql": [
        "idx_incidents_org_idempotency_key",
        "ON incidents(organization_id, idempotency_key)",
        "idx_audit_log_org_dedupe_key",
        "ON audit_log(organization_id, dedupe_key)",
    ],
}

for rel, tokens in checks.items():
    path = root / "apps" / rel if rel.startswith(("backend/", "cve-sync/")) else root / rel
    if not path.exists():
        raise SystemExit(f"background-job idempotency contract: missing {path.relative_to(root)}")
    text = path.read_text()
    for token in tokens:
        if token not in text:
            raise SystemExit(f"background-job idempotency contract: {path.relative_to(root)} missing {token!r}")

worker = (root / "apps/backend/internal/alerts/worker.go").read_text()
if ".BLPop(" in worker or ".BRPop(" in worker:
    raise SystemExit("background-job idempotency contract: destructive blocking list pop reintroduced")

main = (root / "apps/backend/cmd/server/main.go").read_text()
for direct in ("go syncTLEs(",):
    if direct in main:
        raise SystemExit(f"background-job idempotency contract: unlocked scheduled call reintroduced: {direct}")

print("background-job idempotency contract: OK")
