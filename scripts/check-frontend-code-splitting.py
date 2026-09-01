#!/usr/bin/env python3
from pathlib import Path
import importlib.util
import sys

ROOT = Path(__file__).resolve().parents[1]
DASHBOARD = ROOT / "apps/frontend/src/components/Dashboard.jsx"
REPAIR = Path(__file__).with_name("repair-frontend-code-splitting-finalize-v1.py")

spec = importlib.util.spec_from_file_location("frontend_split_finalize", REPAIR)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)


def main():
    if not DASHBOARD.exists():
        print(f"frontend code-splitting contract: ERROR: missing {DASHBOARD}", file=sys.stderr)
        return 2
    text = DASHBOARD.read_text(encoding="utf-8")
    try:
        count = mod.validate_repaired_text(text)
    except ValueError as exc:
        print(f"frontend code-splitting contract: ERROR: {exc}", file=sys.stderr)
        return 1
    if count < 10:
        print(
            f"frontend code-splitting contract: ERROR: only {count} lazy declarations; expected a meaningful split",
            file=sys.stderr,
        )
        return 1
    print(f"frontend code-splitting contract: OK ({count} lazy declarations)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
