#!/usr/bin/env python3
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[1]
server = root / 'apps/backend/cmd/server'
legacy = sorted(p.relative_to(root) for p in server.glob('*.go') if not p.name.endswith('_test.go') and re.search(r'_v\d+\.go$', p.name))
if legacy:
    print('Production Go filenames must not carry implementation-version suffixes:', file=sys.stderr)
    for path in legacy:
        print(f'  - {path}', file=sys.stderr)
    raise SystemExit(1)
print('source naming conventions: ok')
