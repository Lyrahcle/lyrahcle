#!/usr/bin/env python3
"""Consolidate Lyrahcle local DR drill reports into one auditable evidence record."""
from __future__ import annotations

import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def load_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        data = json.load(handle)
    if not isinstance(data, dict):
        raise ValueError(f"{path}: expected JSON object")
    return data


def require_pass(report: dict[str, Any], label: str) -> None:
    if report.get("status") != "PASS":
        raise ValueError(f"{label}: status must be PASS")


def validate_kind(kind: dict[str, Any]) -> list[dict[str, Any]]:
    require_pass(kind, "kind report")
    if kind.get("environment") != "local-kind":
        raise ValueError("kind report: environment must be local-kind")
    results = kind.get("results")
    if not isinstance(results, list) or not results:
        raise ValueError("kind report: non-empty results required")
    checked = []
    for index, result in enumerate(results):
        if not isinstance(result, dict):
            raise ValueError(f"kind result {index}: expected object")
        name = str(result.get("name", "")).strip()
        actual = float(result["actual_seconds"])
        target = float(result["target_seconds"])
        status = result.get("status")
        if not name or actual < 0 or target <= 0:
            raise ValueError(f"kind result {index}: invalid measurement")
        if status != "PASS" or actual > target:
            raise ValueError(f"kind result {name}: target exceeded or status not PASS")
        checked.append({"name": name, "observed_seconds": actual, "target_seconds": target, "status": "PASS"})
    return checked


def validate_restore(restore: dict[str, Any]) -> dict[str, Any]:
    require_pass(restore, "restore report")
    backup_seconds = float(restore["backup_seconds"])
    restore_seconds = float(restore["restore_seconds"])
    tables = int(restore["tables_verified"])
    migrations = int(restore["migrations_verified"])
    if backup_seconds < 0 or restore_seconds < 0 or tables <= 0 or migrations <= 0:
        raise ValueError("restore report: invalid backup/restore measurement")
    return {
        "backup_seconds": backup_seconds,
        "restore_seconds": restore_seconds,
        "tables_verified": tables,
        "migrations_verified": migrations,
        "status": "PASS",
    }


def build_evidence(kind: dict[str, Any], restore: dict[str, Any]) -> dict[str, Any]:
    kind_results = validate_kind(kind)
    restore_result = validate_restore(restore)
    worst_ratio = max(item["observed_seconds"] / item["target_seconds"] for item in kind_results)
    return {
        "schema_version": 1,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "evidence_scope": "local-kind-and-local-postgresql-restore",
        "certification_scope": "CYSAT demo evidence; not an AWS production DR certification",
        "status": "PASS",
        "rto": {
            "kind_measurements": kind_results,
            "worst_target_utilization_percent": round(worst_ratio * 100, 2),
            "restore_seconds": restore_result["restore_seconds"],
        },
        "rpo": {
            "validated_scope": "backup boundary",
            "durable_data_loss_observed": "0 rows at the verified backup boundary",
            "production_limit": "AWS/PITR RPO remains a design objective until an AWS restore drill is executed",
        },
        "restore_integrity": restore_result,
        "source_reports": {
            "kind_generated_at": kind.get("generated_at"),
            "restore_generated_at": restore.get("generated_at"),
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--kind-report", default="/tmp/lyrahcle-k8s-resilience-report.json")
    parser.add_argument("--restore-report", default="/tmp/lyrahcle-dr-restore-report.json")
    parser.add_argument("--output", default="/tmp/lyrahcle-dr-evidence.json")
    args = parser.parse_args()
    try:
        evidence = build_evidence(load_json(Path(args.kind_report)), load_json(Path(args.restore_report)))
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        print(f"DR evidence: FAIL - {exc}")
        return 1
    output = Path(args.output)
    output.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print("DR evidence: PASS")
    print(f"evidence: {output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
