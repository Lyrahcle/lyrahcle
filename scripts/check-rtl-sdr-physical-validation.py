#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.')
checks = {
    'harness': ('scripts/validate-rtl-sdr-physical.py', ['rtl_test', 'rtl_power', 'physical_hardware_observed', 'gate_complete', 'capture_sha256']),
    'guide': ('docs/04-sdr-agent-guide.md', ['RTL-SDR lab evidence harness', 'validate-rtl-sdr-physical.py', 'gate_complete']),
    'roadmap': ('docs/improvement-roadmap.md', ['RTL-SDR evidence harness']),
}
errors = []
for label, (rel, needles) in checks.items():
    path = root / rel
    if not path.exists():
        errors.append(f'{label}: missing {rel}')
        continue
    text = path.read_text(errors='replace')
    for needle in needles:
        if needle not in text:
            errors.append(f'{label}: {rel} missing {needle!r}')
if errors:
    print('\n'.join(errors), file=sys.stderr)
    raise SystemExit(1)
print('RTL-SDR physical validation harness contract: OK')
