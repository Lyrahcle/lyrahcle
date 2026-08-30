#!/usr/bin/env python3
"""Regression contract for dynamically constructed SQL in backend stores."""
from __future__ import annotations
import pathlib
import re
import sys

ROOT = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
STORE = ROOT / "apps" / "backend" / "internal" / "store"
DYNAMIC_SQL_ALLOWLIST = {"space_cves.go"}
SQL_WORD = re.compile(r"\b(?:SELECT|INSERT|UPDATE|DELETE|ORDER\s+BY|WHERE)\b", re.I)


def quoted_literals(text: str):
    i = 0
    while i < len(text):
        q = text[i]
        if q not in {'"', '`'}:
            i += 1
            continue
        start = i
        i += 1
        escaped = False
        while i < len(text):
            ch = text[i]
            if q == '"' and escaped:
                escaped = False
                i += 1
                continue
            if q == '"' and ch == '\\':
                escaped = True
                i += 1
                continue
            if ch == q:
                yield start, i + 1, text[start + 1:i]
                i += 1
                break
            i += 1


def has_dynamic_sql(text: str) -> bool:
    # SQL literal used as the format string of fmt.Sprintf.
    for m in re.finditer(r"fmt\.Sprintf\s*\(\s*", text):
        after = text[m.end():]
        literals = list(quoted_literals(after[:4000]))
        if literals and literals[0][0] == 0 and SQL_WORD.search(literals[0][2]):
            return True
    # SQL literal immediately concatenated with runtime text.
    for start, end, literal in quoted_literals(text):
        if not SQL_WORD.search(literal):
            continue
        if re.match(r"\s*\+", text[end:]):
            return True
    return False

errors = []
for path in STORE.glob("*.go"):
    if path.name.endswith("_test.go"):
        continue
    text = path.read_text(encoding="utf-8")
    if has_dynamic_sql(text) and path.name not in DYNAMIC_SQL_ALLOWLIST:
        errors.append(f"{path.name}: dynamic SQL construction requires security review")

approved = STORE / "space_cves.go"
if approved.exists():
    text = approved.read_text(encoding="utf-8")
    required = [
        "validSpaceCVESortFields",
        "spaceCVESortClause(filter.SortBy, filter.SortDesc)",
        "args = append(args, v)",
        'return fmt.Sprintf("$%d", len(args))',
    ]
    for fragment in required:
        if fragment not in text:
            errors.append(f"space_cves.go: reviewed SQL safety contract changed; missing {fragment!r}")

if errors:
    print("SQL construction contract: FAILED", file=sys.stderr)
    for error in errors:
        print(f" - {error}", file=sys.stderr)
    raise SystemExit(1)
print("SQL construction contract: OK")
