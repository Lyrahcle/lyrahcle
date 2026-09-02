#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
required = {
    'apps/frontend/src/components/ToastProvider.tsx': [
        "export type ToastTone = 'success' | 'info' | 'warning' | 'error'",
        'const MAX_TOASTS = 4',
        "tone === 'error' || tone === 'warning' ? 0 : 6000",
        "role={urgent ? 'alert' : 'status'}",
        "aria-live={urgent ? 'assertive' : 'polite'}",
        'dedupeKey',
    ],
    'apps/frontend/src/components/StatusState.tsx': [
        "export type StatusStateKind = 'empty' | 'loading' | 'error' | 'offline' | 'forbidden'",
        "role={isError ? 'alert' : isLoading ? 'status' : undefined}",
        'aria-busy={isLoading || undefined}',
        'data-state-kind={kind}',
    ],
    'apps/frontend/src/components/SkipLink.tsx': ['href="#main-content"', "t('ux.skipToContent')"],
    'apps/frontend/src/main.jsx': ['<ToastProvider>', '<App />', '</ToastProvider>'],
    'apps/frontend/src/components/Dashboard.tsx': ['<SkipLink />', '<main id="main-content"', 'canReadUsers={usersRead || admin}'],
    'apps/frontend/src/components/RBACPanel.tsx': ['useToast()', "kind=\"empty\"", "kind=\"forbidden\"", 'canReadUsers?: boolean', "t('ux.notifications.roleCreated')"],
    'apps/frontend/src/components/RateLimitPolicyPanel.tsx': ['<StatusState', "kind={offline ? 'offline' : 'error'}", "actionLabel={t('ux.states.retry')}"],
    'apps/frontend/src/components/AssetAlertsPanel.jsx': ['<StatusState kind="empty"', 'useToast()'],
    'apps/frontend/src/components/IncidentsPanel.tsx': ['<StatusState kind="empty"'],
    'apps/frontend/src/components/ThreatIntelPanel.jsx': ['<StatusState kind="empty"'],
    'apps/frontend/src/index.css': [':focus-visible', 'outline: 2px solid rgb(var(--lyra-gold-bright))'],
}
errors = []
for relative, markers in required.items():
    path = ROOT / relative
    if not path.is_file():
        errors.append(f'missing {relative}')
        continue
    source = path.read_text()
    for marker in markers:
        if marker not in source:
            errors.append(f'{relative} missing {marker}')
translations = ROOT / 'apps/frontend/src/lib/translations.js'
if translations.is_file():
    source = translations.read_text()
    if 'const uxFoundationTranslations = {' not in source:
        errors.append('UX translations are missing')
    for language in ('en:', 'es:', 'fr:', 'de:'):
        if language not in source[source.find('const uxFoundationTranslations = {'):]:
            errors.append(f'UX translations missing language {language[:-1]}')
else:
    errors.append('missing translations.js')
if errors:
    print('frontend UX foundation contract: ERROR')
    print('\n'.join(errors))
    sys.exit(1)
print('frontend UX foundation contract: OK')
