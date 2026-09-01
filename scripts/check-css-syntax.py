#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
violations = []

for path in sorted((ROOT / 'apps' / 'frontend' / 'src').rglob('*.css')):
    text = path.read_text(errors='ignore')
    depth = 0
    quote = None
    escaped = False
    line = 1
    for char in text:
        if char == '\n':
            line += 1
        if quote:
            if escaped:
                escaped = False
            elif char == '\\':
                escaped = True
            elif char == quote:
                quote = None
            continue
        if char in {'\"', "'"}:
            quote = char
            continue
        if char == '{':
            depth += 1
        elif char == '}':
            depth -= 1
            if depth < 0:
                violations.append(f'{path.relative_to(ROOT)}:{line}: unexpected closing brace')
                depth = 0
    if quote:
        violations.append(f'{path.relative_to(ROOT)}:{line}: unterminated string')
    if depth:
        violations.append(f'{path.relative_to(ROOT)}: unclosed block depth {depth}')

if violations:
    print('CSS syntax contract: ERROR')
    print('\n'.join(violations))
    sys.exit(1)

print('CSS syntax contract: OK')
