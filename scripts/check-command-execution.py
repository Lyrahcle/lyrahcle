#!/usr/bin/env python3
"""Fail CI if application source introduces unsafe subprocess/shell execution.

Lyrahcle does not need to spawn operating-system commands at runtime. Keeping this
as an explicit invariant removes an entire command-injection class. Operational
shell scripts under scripts/ and container entrypoints are intentionally outside
this contract; application source under apps/ and packages/ is covered.
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

SOURCE_SUFFIXES = {".go", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"}
SKIP_DIRS = {"node_modules", "vendor", "dist", "build", ".git", "coverage"}

RULES: tuple[tuple[str, re.Pattern[str]], ...] = (
    ("Go os/exec import", re.compile(r'["`]os/exec["`]')),
    ("Go process execution", re.compile(r"\b(?:exec\.Command(?:Context)?|os\.StartProcess|syscall\.Exec)\s*\(")),
    ("Node child_process import", re.compile(r"(?:from\s+[\"'](?:node:)?child_process[\"']|require\s*\(\s*[\"'](?:node:)?child_process[\"']\s*\))")),
    ("Node process execution", re.compile(r"\b(?:exec|execFile|execSync|execFileSync|spawn|spawnSync)\s*\(")),
    ("shell -c invocation", re.compile(r"(?:/bin/)?(?:ba)?sh\s+[^\n]*?-c\b")),
)


def iter_sources(root: Path):
    for top in (root / "apps", root / "packages"):
        if not top.exists():
            continue
        for path in top.rglob("*"):
            if not path.is_file() or path.suffix not in SOURCE_SUFFIXES:
                continue
            if any(part in SKIP_DIRS for part in path.parts):
                continue
            yield path


def scan(root: Path) -> list[str]:
    findings: list[str] = []
    for path in iter_sources(root):
        try:
            text = path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue
        for line_no, line in enumerate(text.splitlines(), 1):
            for label, pattern in RULES:
                if pattern.search(line):
                    findings.append(f"{path.relative_to(root)}:{line_no}: {label}: {line.strip()}")
    return findings


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("root", nargs="?", default=".", help="repository root")
    args = parser.parse_args(argv)
    root = Path(args.root).resolve()

    findings = scan(root)
    if findings:
        print("Unsafe runtime command execution is forbidden in Lyrahcle application source:", file=sys.stderr)
        for finding in findings:
            print(f"  {finding}", file=sys.stderr)
        print("Use an in-process library or a narrowly reviewed adapter instead of a shell/subprocess.", file=sys.stderr)
        return 1

    print("command-execution contract: OK (no runtime subprocess/shell APIs found)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
