#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
checks = {
    'apps/backend/internal/keyprovider/provider.go': [
        'type Provider interface', 'ActiveSigningKey(context.Context)', 'SigningKey(context.Context, string)',
        'type SigningKey interface', 'Sign(context.Context, []byte)', 'func Ed25519KID(',
    ],
    'apps/backend/internal/agentconfig/signing.go': [
        'provider keyprovider.Provider', 's.provider.ActiveSigningKey(ctx)', 'key.Sign(ctx, payloadBytes)', 'KID:',
    ],
    'apps/backend/internal/store/agent_config.go': [
        'SigningKID string', 'signing_kid, created_by', 'v.signing_kid',
    ],
    'apps/backend/migrations/0036_agent_config_signing_kid.sql': [
        'ADD COLUMN IF NOT EXISTS signing_kid', 'ALTER COLUMN signing_kid DROP DEFAULT',
    ],
    'apps/sdr-agent/internal/uplink/uplink.go': ['KID       string `json:"kid,omitempty"`'],
    'apps/sdr-agent/internal/remoteconfig/remoteconfig.go': [
        'VerificationKey(envelope.KID)', 'LegacyKey()', 'unknown signing key id',
    ],
    'apps/sdr-agent/internal/remoteconfig/keys.go': [
        'VerificationKeyProvider', 'os.ModeSymlink', 'ed25519KID(',
    ],
    'apps/frontend/src/components/AgentRemoteConfigPanel.jsx': [
        'current.signing_kid', 'item.signing_kid', "remoteSigningKey",
    ],
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

signer = (root / 'apps/backend/internal/agentconfig/signing.go').read_text(encoding='utf-8')
for forbidden in ('x509.ParsePKCS8PrivateKey(', 'os.ReadFile(', 'ed25519.Sign('):
    if forbidden in signer:
        errors.append(f'agent config signer bypasses KeyProvider via {forbidden}')

migration = (root / 'apps/backend/migrations/0036_agent_config_signing_kid.sql').read_text(encoding='utf-8')
if "signing_kid = ''" not in migration:
    errors.append('migration must preserve an explicit empty legacy marker for pre-KID rows')

if errors:
    print('keyprovider/KID contract: FAILED')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)
print('keyprovider/KID contract: OK')
