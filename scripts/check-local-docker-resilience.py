#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
errors = []
compose = (ROOT / 'docker-compose.yml').read_text(encoding='utf-8')
for token in ['image: lyrahcle/backend:local', 'image: lyrahcle/frontend:local', 'image: lyrahcle/cve-sync:local']:
    if token not in compose:
        errors.append(f'docker-compose.yml missing {token}')
for name in ['deploy/docker/Dockerfile.backend', 'deploy/docker/Dockerfile.frontend']:
    text = (ROOT / name).read_text(encoding='utf-8')
    if '# syntax=docker/dockerfile:1' in text:
        errors.append(f'{name} still forces remote dockerfile frontend resolution')
script = ROOT / 'scripts/docker-local-up.sh'
if not script.exists():
    errors.append('scripts/docker-local-up.sh missing')
else:
    text = script.read_text(encoding='utf-8')
    for token in ['docker compose "$@" build', '--no-build --pull never', 'docker compose "$@" config --images', 'registry-1\\.docker\\.io', 'lyrahcle-main-backend:latest']:
        if token not in text:
            errors.append(f'docker-local-up.sh missing {token}')
if errors:
    print('local docker resilience contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)
print('local docker resilience contract: OK')
