#!/usr/bin/env python3
"""Verify that every Lyrahcle Go build surface uses the pinned toolchain."""
from __future__ import annotations

import re
import sys
from pathlib import Path

PINNED_GO = "1.26.7"
GO_LANGUAGE = "1.26.0"
MODULES = (
    Path("apps/backend/go.mod"),
    Path("apps/cve-sync/go.mod"),
    Path("apps/sdr-agent/go.mod"),
)
DOCKERFILES = (
    Path("deploy/docker/Dockerfile.backend"),
    Path("deploy/docker/Dockerfile.cve-sync"),
    Path("deploy/docker/Dockerfile.sdr-agent"),
)
WORKFLOWS = (
    Path(".github/workflows/ci.yml"),
    Path(".github/workflows/release.yml"),
)


def check(root: Path) -> list[str]:
    errors: list[str] = []

    for rel in MODULES:
        path = root / rel
        text = path.read_text(encoding="utf-8")
        go_match = re.search(r"(?m)^go\s+(\S+)\s*$", text)
        toolchain_match = re.search(r"(?m)^toolchain\s+go(\S+)\s*$", text)
        if not go_match or go_match.group(1) != GO_LANGUAGE:
            errors.append(f"{rel}: go directive must be {GO_LANGUAGE}")
        if not toolchain_match or toolchain_match.group(1) != PINNED_GO:
            errors.append(f"{rel}: toolchain must be go{PINNED_GO}")

    expected_image = f"golang:{PINNED_GO}-alpine"
    for rel in DOCKERFILES:
        path = root / rel
        text = path.read_text(encoding="utf-8")
        first_go = next((line.strip() for line in text.splitlines() if line.startswith("FROM golang:")), "")
        if not first_go.startswith(f"FROM {expected_image}"):
            errors.append(f"{rel}: builder must use {expected_image}")

    for rel in WORKFLOWS:
        path = root / rel
        text = path.read_text(encoding="utf-8")
        versions = re.findall(r"go-version:\s*['\"]?([^'\"\s]+)", text)
        if not versions:
            errors.append(f"{rel}: no setup-go version found")
            continue
        for version in versions:
            if version != PINNED_GO:
                errors.append(f"{rel}: setup-go uses {version}, expected {PINNED_GO}")
        if "go-version: 'stable'" in text or 'go-version: "stable"' in text:
            errors.append(f"{rel}: floating stable Go version is forbidden")

    return errors


def main() -> int:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors = check(root)
    if errors:
        for error in errors:
            print(f"go-toolchain contract: ERROR: {error}", file=sys.stderr)
        return 1
    print(f"go-toolchain contract: OK (Go {PINNED_GO})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
