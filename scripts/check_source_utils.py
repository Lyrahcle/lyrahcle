#!/usr/bin/env python3
"""Small helpers for contract checks that span modular source files."""

from pathlib import Path


def read_source_tree(directory: Path, pattern: str = '*.go', include_tests: bool = False) -> str:
    """Return stable concatenated source text for a directory-level contract check."""
    if not directory.exists():
        return ''
    files = sorted(directory.glob(pattern))
    if not include_tests:
        files = [path for path in files if not path.name.endswith('_test.go')]
    return '\n'.join(path.read_text(encoding='utf-8', errors='replace') for path in files if path.is_file())
