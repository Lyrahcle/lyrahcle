#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
SUFFIXES = {'.go', '.js', '.jsx', '.py', '.sh', '.yaml', '.yml', '.css', '.hcl', '.tf', '.ts', '.tsx'}
SKIP = {'.git', 'node_modules', 'dist', 'coverage', 'vendor'}
violations = []
for path in ROOT.rglob('*'):
    if not path.is_file() or path.suffix.lower() not in SUFFIXES or any(part in SKIP for part in path.parts):
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
