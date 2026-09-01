#!/usr/bin/env python3
"""Enforce Lyrahcle frontend JavaScript chunk budgets after a Vite build."""
from __future__ import annotations

import argparse
import gzip
import os
import sys
from dataclasses import dataclass
from pathlib import Path

DEFAULT_MAX_JS_KIB = 500
DEFAULT_MAX_GZIP_KIB = 180
DEFAULT_WARN_JS_KIB = 450
DEFAULT_MIN_JS_CHUNKS = 5


@dataclass(frozen=True)
class AssetSize:
    path: Path
    raw_bytes: int
    gzip_bytes: int

    @property
    def raw_kib(self) -> float:
        return self.raw_bytes / 1024

    @property
    def gzip_kib(self) -> float:
        return self.gzip_bytes / 1024


def javascript_assets(dist_dir: Path) -> list[Path]:
    assets_dir = dist_dir / "assets"
    if not assets_dir.is_dir():
        raise FileNotFoundError(
            f"frontend assets directory not found: {assets_dir}; run `npm run build` first"
        )
    return sorted(
        p for p in assets_dir.glob("*.js")
        if p.is_file() and not p.name.endswith(".map.js")
    )


def measure(path: Path) -> AssetSize:
    payload = path.read_bytes()
    return AssetSize(
        path=path,
        raw_bytes=len(payload),
        gzip_bytes=len(gzip.compress(payload, mtime=0)),
    )


def env_int(name: str, default: int) -> int:
    raw = os.environ.get(name)
    if raw is None or raw == "":
        return default
    try:
        value = int(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be an integer, got {raw!r}") from exc
    if value <= 0:
        raise ValueError(f"{name} must be greater than zero")
    return value


def evaluate(
    sizes: list[AssetSize],
    *,
    max_js_kib: int,
    max_gzip_kib: int,
    warn_js_kib: int,
    min_js_chunks: int,
) -> tuple[list[str], list[str]]:
    errors: list[str] = []
    warnings: list[str] = []

    if len(sizes) < min_js_chunks:
        errors.append(
            f"only {len(sizes)} JavaScript chunks found; expected at least "
            f"{min_js_chunks} after code splitting"
        )

    for asset in sizes:
        label = asset.path.name
        if asset.raw_kib > max_js_kib:
            errors.append(
                f"{label}: {asset.raw_kib:.2f} KiB exceeds raw JS budget "
                f"{max_js_kib} KiB"
            )
        if asset.gzip_kib > max_gzip_kib:
            errors.append(
                f"{label}: {asset.gzip_kib:.2f} KiB exceeds gzip budget "
                f"{max_gzip_kib} KiB"
            )
        if asset.raw_kib > warn_js_kib:
            warnings.append(
                f"{label}: {asset.raw_kib:.2f} KiB is close to the "
                f"{max_js_kib} KiB hard limit"
            )
    return errors, warnings


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "dist_dir",
        nargs="?",
        default="apps/frontend/dist",
        help="Vite dist directory (default: apps/frontend/dist)",
    )
    parser.add_argument("--max-js-kib", type=int, default=None)
    parser.add_argument("--max-gzip-kib", type=int, default=None)
    parser.add_argument("--warn-js-kib", type=int, default=None)
    parser.add_argument("--min-js-chunks", type=int, default=None)
    parser.add_argument(
        "--require-build",
        action="store_true",
        help="fail if the Vite build/assets are missing instead of skipping",
    )
    args = parser.parse_args(argv)

    try:
        max_js_kib = args.max_js_kib or env_int(
            "FRONTEND_MAX_JS_KIB", DEFAULT_MAX_JS_KIB
        )
        max_gzip_kib = args.max_gzip_kib or env_int(
            "FRONTEND_MAX_GZIP_KIB", DEFAULT_MAX_GZIP_KIB
        )
        warn_js_kib = args.warn_js_kib or env_int(
            "FRONTEND_WARN_JS_KIB", DEFAULT_WARN_JS_KIB
        )
        min_js_chunks = args.min_js_chunks or env_int(
            "FRONTEND_MIN_JS_CHUNKS", DEFAULT_MIN_JS_CHUNKS
        )
        if warn_js_kib >= max_js_kib:
            raise ValueError("FRONTEND_WARN_JS_KIB must be lower than the hard JS budget")
        assets = javascript_assets(Path(args.dist_dir))
    except ValueError as exc:
        print(f"frontend bundle budget: ERROR: {exc}", file=sys.stderr)
        return 2
    except FileNotFoundError as exc:
        if args.require_build:
            print(f"frontend bundle budget: ERROR: {exc}", file=sys.stderr)
            return 2
        print(f"frontend bundle budget: SKIP: {exc}")
        return 0

    if not assets:
        if args.require_build:
            print("frontend bundle budget: ERROR: no JavaScript assets found", file=sys.stderr)
            return 2
        print(
            "frontend bundle budget: SKIP: no JavaScript assets found; "
            "run after the frontend build for enforcement"
        )
        return 0

    sizes = [measure(path) for path in assets]
    errors, warnings = evaluate(
        sizes,
        max_js_kib=max_js_kib,
        max_gzip_kib=max_gzip_kib,
        warn_js_kib=warn_js_kib,
        min_js_chunks=min_js_chunks,
    )

    print("Frontend bundle assets:")
    for asset in sorted(sizes, key=lambda item: item.raw_bytes, reverse=True):
        print(
            f"  {asset.path.name}: {asset.raw_kib:.2f} KiB raw / "
            f"{asset.gzip_kib:.2f} KiB gzip"
        )

    for warning in warnings:
        print(f"WARNING: {warning}", file=sys.stderr)

    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        print("frontend bundle budget: FAILED", file=sys.stderr)
        return 1

    largest = max(sizes, key=lambda item: item.raw_bytes)
    print(
        "frontend bundle budget: OK "
        f"({len(sizes)} chunks; largest {largest.raw_kib:.2f} KiB raw / "
        f"{largest.gzip_kib:.2f} KiB gzip; budgets "
        f"{max_js_kib}/{max_gzip_kib} KiB)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
