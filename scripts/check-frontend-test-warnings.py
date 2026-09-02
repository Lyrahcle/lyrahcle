#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

PATTERNS = [
    ("react act warning", re.compile(r"not wrapped in act\(", re.IGNORECASE)),
    ("unhandled promise rejection", re.compile(r"unhandled(?: promise)? rejection", re.IGNORECASE)),
    ("unhandled test error", re.compile(r"unhandled errors?", re.IGNORECASE)),
]


def check(text: str) -> list[str]:
    found: list[str] = []
    for label, pattern in PATTERNS:
        if pattern.search(text) and label not in found:
            found.append(label)
    return found


def read_input(path_arg: str | None) -> tuple[str | None, str | None]:
    """Return (text, skip_reason).

    Contract/parity runners invoke check scripts with the repository root (`.`).
    A directory is not captured Vitest output, so directory mode is an intentional
    no-op. Enforcement remains strict when a captured output file or stdin is used.
    """
    if path_arg is None:
        return sys.stdin.read(), None

    path = Path(path_arg)
    if path.is_dir():
        return None, f"{path_arg} is a directory; warning enforcement requires captured Vitest output"
    if not path.exists():
        raise FileNotFoundError(path)
    return path.read_text(encoding="utf-8", errors="replace"), None


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Fail on unexpected React test warnings")
    parser.add_argument("path", nargs="?", help="Captured Vitest output; stdin when omitted")
    args = parser.parse_args(argv)

    try:
        text, skip_reason = read_input(args.path)
    except OSError as exc:
        print(f"frontend test warnings: ERROR: cannot read input: {exc}", file=sys.stderr)
        return 2

    if skip_reason is not None:
        print(f"frontend test warnings: SKIP: {skip_reason}")
        return 0

    assert text is not None
    failures = check(text)
    if failures:
        for item in failures:
            print(f"frontend test warnings: ERROR: {item} detected", file=sys.stderr)
        return 1

    print("frontend test warnings: OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
