#!/usr/bin/env python3
"""Regression contract: every tenant-owned table must have a conscious RLS
decision (database-enforced Row Level Security, or a documented app-level
exception), so a future migration cannot silently add an `organization_id`
column without either enabling Postgres RLS on it or explicitly recording
why it relies on application-level tenant scoping instead.

This does not change runtime behavior. It only fails CI when the set of
tenant-owned tables and the set of RLS-covered/exempted tables drift apart,
so the drift is a code-review decision instead of a silent gap.
"""
from __future__ import annotations
import pathlib
import re
import sys

ROOT = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
MIGRATIONS = ROOT / "apps" / "backend" / "migrations"

CREATE_TABLE_RE = re.compile(
    r"CREATE TABLE(?: IF NOT EXISTS)? (?P<name>[a-z_]+)\s*\((?P<body>.*?)\n\);",
    re.IGNORECASE | re.DOTALL,
)
ADD_COLUMN_RE = re.compile(
    r"ALTER TABLE (?P<name>[a-z_]+) ADD COLUMN(?: IF NOT EXISTS)? organization_id",
    re.IGNORECASE,
)
ENABLE_RLS_RE = re.compile(
    r"ALTER TABLE (?P<name>[a-z_]+) ENABLE ROW LEVEL SECURITY",
    re.IGNORECASE,
)
# RLS is also enabled dynamically for a batch of tables via
# `FOREACH target_table IN ARRAY ARRAY[ 'a', 'b', ... ] LOOP ... EXECUTE
# format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', target_table) ...`.
DYNAMIC_RLS_MARKER = "ENABLE ROW LEVEL SECURITY', target_table"
RLS_ARRAY_RE = re.compile(r"ARRAY\s*\[(?P<items>(?:\s*'[a-z_]+'\s*,?)+)\]", re.IGNORECASE)
RLS_ARRAY_ENTRY_RE = re.compile(r"'([a-z_]+)'")

# Tables that own an `organization_id` column but are deliberately protected
# by application-level tenant scoping (see internal/store/tenant_scope.go
# and the mTLS/JWT-bound query paths) rather than Postgres RLS. Each entry
# must carry a one-line rationale; removing a table here without adding a
# matching `ENABLE ROW LEVEL SECURITY` migration will fail this contract.
APP_LEVEL_TENANT_EXCEPTIONS = {
    "users": "authenticated via JWT claims + org-scoped lookups, not the HTTP RLS session GUC",
    "agent_identities": "agent identity resolved from mTLS client cert, not the human RLS session context",
    "agent_keys": "child of agent_identities; scoped through the parent's app-level tenant check",
    "agent_certificates": "child of agent_identities; scoped through the parent's app-level tenant check",
    "agent_enrollment_tokens": "single-use pre-identity tokens; no authenticated org session exists yet",
    "agent_config_state": "written by the agent's mTLS-authenticated config channel, not the HTTP RLS path",
    "agent_config_versions": "written by the agent's mTLS-authenticated config channel, not the HTTP RLS path",
    "agent_config_acks": "written by the agent's mTLS-authenticated config channel, not the HTTP RLS path",
}


def tenant_owned_tables(migrations_text: dict[str, str]) -> dict[str, str]:
    """Return {table_name: migration_file} for every table with a direct
    organization_id column, keeping the migration where it first appears."""
    owners: dict[str, str] = {}
    for fname, text in migrations_text.items():
        for match in CREATE_TABLE_RE.finditer(text):
            if "organization_id" in match.group("body"):
                owners.setdefault(match.group("name"), fname)
        for match in ADD_COLUMN_RE.finditer(text):
            owners.setdefault(match.group("name"), fname)
    return owners


def rls_covered_tables(migrations_text: dict[str, str]) -> set[str]:
    covered: set[str] = set()
    for text in migrations_text.values():
        for match in ENABLE_RLS_RE.finditer(text):
            covered.add(match.group("name"))
        if DYNAMIC_RLS_MARKER in text:
            for array_match in RLS_ARRAY_RE.finditer(text):
                covered.update(RLS_ARRAY_ENTRY_RE.findall(array_match.group("items")))
    return covered


def main() -> int:
    if not MIGRATIONS.exists():
        print(f"check-rls-coverage: FAIL - migrations directory not found: {MIGRATIONS}")
        return 1

    migrations_text = {
        p.name: p.read_text(encoding="utf-8", errors="replace")
        for p in sorted(MIGRATIONS.glob("*.sql"))
    }

    owners = tenant_owned_tables(migrations_text)
    covered = rls_covered_tables(migrations_text)

    problems = []
    for table, source_migration in sorted(owners.items()):
        if table in covered:
            continue
        if table in APP_LEVEL_TENANT_EXCEPTIONS:
            continue
        problems.append(
            f"{table} (added in {source_migration}) has organization_id but no "
            f"'ALTER TABLE {table} ENABLE ROW LEVEL SECURITY' migration and is not "
            f"in APP_LEVEL_TENANT_EXCEPTIONS with a documented rationale"
        )

    # Catch the opposite drift too: an exception on record for a table that
    # no longer exists, or that gained real RLS since the exception was written.
    stale_exceptions = [
        t for t in APP_LEVEL_TENANT_EXCEPTIONS if t not in owners or t in covered
    ]
    for table in stale_exceptions:
        problems.append(
            f"{table} is listed in APP_LEVEL_TENANT_EXCEPTIONS but is either no "
            f"longer a tenant-owned table or now has RLS - remove the stale exception"
        )

    if problems:
        print("RLS coverage contract: FAIL")
        for item in problems:
            print(" -", item)
        return 1

    print(
        f"RLS coverage contract: OK "
        f"({len(covered)} tables RLS-enforced, {len(APP_LEVEL_TENANT_EXCEPTIONS)} "
        f"documented app-level exceptions, {len(owners)} tenant-owned tables total)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
