#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
palette = (ROOT / 'apps/frontend/src/components/CommandPalette.jsx').read_text()
header = (ROOT / 'apps/frontend/src/components/Header.jsx').read_text()
dashboard = (ROOT / 'apps/frontend/src/components/Dashboard.jsx').read_text()
translations = (ROOT / 'apps/frontend/src/lib/translations.js').read_text()
required = [
    ('mount command palette in header', "import CommandPalette from './CommandPalette.jsx'" in header and '<CommandPalette' in header),
    ('support Ctrl/Cmd+K', 'event.metaKey || event.ctrlKey' in palette and "event.key.toLocaleLowerCase() === 'k'" in palette),
    ('support keyboard result navigation', "event.key === 'ArrowDown'" in palette and "event.key === 'ArrowUp'" in palette and "event.key === 'Enter'" in palette),
    ('reuse global search API', "import { globalSearch } from '../lib/api.js'" in palette and 'await globalSearch(value, controller.signal)' in palette),
    ('filter search results to allowed destinations', '.filter((result) => allowedTabs.has(result.tab))' in palette),
    ('open satellite profile directly', "result?.type === 'satellites'" in dashboard and 'setSelectedSatelliteId(result.id)' in dashboard),
    ('preserve permission-aware navigation source', 'navigationItems={tabs}' in dashboard),
    ('provide multilingual command copy', 'commandPaletteTranslations' in translations and 'Paleta de comandos' in translations),
    ('use accessible dialog and listbox', 'role="dialog"' in palette and 'role="listbox"' in palette and 'role="option"' in palette),
]
failed = [name for name, ok in required if not ok]
if failed:
    print('command palette contract: ERROR: ' + ', '.join(failed))
    sys.exit(1)
print('command palette contract: OK')
