#!/usr/bin/env python3
import os
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
SUFFIXES = {'.go', '.js', '.jsx', '.py', '.sh', '.yaml', '.yml', '.css', '.hcl', '.tf', '.ts', '.tsx'}
SKIP = {
    '.git',
    '.terraform',
    '.terragrunt-cache',
    '.venv',
    'venv',
    'node_modules',
    'dist',
    'build',
    'coverage',
    'vendor',
}
violations = []
for directory, dirnames, filenames in os.walk(ROOT):
    dirnames[:] = [name for name in dirnames if name not in SKIP]
    base = Path(directory)
    for filename in filenames:
        path = base / filename
        if path.suffix.lower() not in SUFFIXES:
            continue
        for number, line in enumerate(path.read_text(errors='ignore').splitlines(), 1):
            stripped = line.lstrip()
            if stripped.startswith('//go:') or stripped.startswith('// +build') or stripped.startswith('/// <reference ') or stripped.startswith('#!'):
                continue
            if stripped.startswith('//') or stripped.startswith('/*') or stripped.startswith('<!--') or stripped.startswith('# '):
                violations.append(f'{path.relative_to(ROOT)}:{number}: {stripped[:80]}')
if violations:
    print('code cleanliness contract: ERROR')
    print('\n'.join(violations[:80]))
    sys.exit(1)
print('code cleanliness contract: OK')
