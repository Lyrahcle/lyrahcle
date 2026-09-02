#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
checks = {
    'apps/backend/internal/keyprovider/provider.go': [
        'type LifecycleProvider interface', 'KeyStatusStaged', 'KeyStatusActive', 'KeyStatusGrace', 'KeyStatusRevoked',
        'PrepareRotation(context.Context)', 'Activate(context.Context, string, time.Duration)', 'Revoke(context.Context, string)',
        'PublicKeyPEM(context.Context, string)', 'managed key directory must be a real directory',
    ],
    'apps/backend/cmd/server/signing_keys_v136.go': [
        '/signing-keys/agent-config', 'platform administrator required', 'ActionSigningKeyPrepared',
        'ActionSigningKeyActivated', 'ActionSigningKeyRevoked', 'public_key_pem',
    ],
    'apps/backend/cmd/server/agent_config_v91.go': [
        'desired config signing key is no longer trusted', 'signingKeys.SigningKey',
    ],
    'apps/frontend/src/components/SigningKeysPanel.jsx': [
        "t('signingKeys.prepare')", 'prepareAgentConfigSigningKeyRotation', 'activateAgentConfigSigningKey',
        'revokeAgentConfigSigningKey', 'publicKey.pem',
    ],
    'apps/frontend/src/components/Dashboard.tsx': ['SigningKeysPanel'],
}
errors = []
for rel, needles in checks.items():
    path = root / rel
    if not path.is_file():
        errors.append(f'missing {rel}')
        continue
    text = path.read_text(encoding='utf-8')
    for needle in needles:
        if needle not in text:
            errors.append(f'{rel}: missing contract token {needle!r}')

ui = (root / 'apps/frontend/src/components/SigningKeysPanel.jsx').read_text(encoding='utf-8')
if 'PRIVATE KEY' in ui:
    errors.append('SigningKeysPanel must never render private key material')

if errors:
    print('signing-key lifecycle contract: FAILED')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)
print('signing-key lifecycle contract: OK')
