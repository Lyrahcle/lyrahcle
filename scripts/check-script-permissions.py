#!/usr/bin/env python3
from __future__ import annotations
import os
import sys
from pathlib import Path

ROOT = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
REQUIRED = (
    'scripts/test-local-v1.sh',
    'scripts/smoke-test.sh',
    'scripts/generate-sbom.sh',
    'scripts/backup-db.sh',
    'scripts/restore-db.sh',
    'scripts/verify-backup.sh',
    'scripts/load-test.sh',
    'scripts/generate-agent-ca.sh',
    'scripts/generate-agent-config-key.sh',
    'scripts/generate-dev-certs.sh',
    'scripts/aws-drift-check.sh',
    'scripts/k8s-enforce-restricted.sh',
    'scripts/aws-build-push-images.sh',
    'scripts/aws-production-readiness.sh',
)
errors=[]
for rel in REQUIRED:
    path=ROOT/rel
    if not path.is_file():
        errors.append(f'{rel}: missing')
        continue
    if not os.access(path, os.X_OK):
        errors.append(f'{rel}: not executable')
if errors:
    print('script permission contract: FAILED', file=sys.stderr)
    for err in errors:
        print(f' - {err}', file=sys.stderr)
    raise SystemExit(1)
print('script permission contract: OK')
