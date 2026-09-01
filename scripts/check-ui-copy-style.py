#!/usr/bin/env python3
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
violations = []
for path in (ROOT / 'apps/frontend/src').rglob('*.jsx'):
    text = path.read_text()
    for number, line in enumerate(text.splitlines(), 1):
        if re.search(r'(?<![A-Za-z-])uppercase(?![A-Za-z-])', line):
            violations.append(f'{path.relative_to(ROOT)}:{number}: forced uppercase class')
if violations:
    print('UI copy style contract: ERROR')
    print('\n'.join(violations[:80]))
    sys.exit(1)
print('UI copy style contract: OK')
