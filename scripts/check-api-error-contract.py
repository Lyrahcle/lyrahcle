#!/usr/bin/env python3
from pathlib import Path
import re
import sys

root = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
server = root / "apps/backend/cmd/server"
leaks = []

for path in sorted(server.glob("*.go")):
    if path.name.endswith("_test.go"):
        continue
    source = path.read_text(encoding="utf-8")
    pattern = re.compile(
        r"http\.Error\([^\n]*\.Error\(\)[^\n]*(?:StatusInternalServerError|,\s*500\s*\))"
    )
    for match in pattern.finditer(source):
        line = source.count("\n", 0, match.start()) + 1
        leaks.append(f"{path.relative_to(root)}:{line}")

if leaks:
    print("internal errors must not be written directly to HTTP responses:", file=sys.stderr)
    for leak in leaks:
        print(f"  - {leak}", file=sys.stderr)
    raise SystemExit(1)

print("API error disclosure contract: ok")
