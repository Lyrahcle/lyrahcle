#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
checks = {
    "apps/backend/internal/satnogs/client.go": [
        '"https://db.satnogs.org/api"',
        '"https://network.satnogs.org/api"',
        '"norad_cat_id"',
        '"external_community"',
        '"radio_activity_context_only"',
        'strings.EqualFold(observation.Status, "future")',
        'receptionFreshness',
    ],
    "apps/backend/cmd/server/routes_satellites_tmtc.go": [
        'Get("/{satelliteID}/external-sources"',
        'satnogs.NewClient(10 * time.Second)',
        'cache.Set(r.Context(), cacheKey, result, 15*time.Minute)',
    ],
    "apps/frontend/src/components/SatelliteSecurityProfile.jsx": [
        'fetchSatelliteExternalSources',
        'satelliteSources.contextNote',
        'network.satnogs.org/observations/',
        '<EvidenceBadge kind="external" />',
    ],
    "apps/backend/cmd/server/openapi.yaml": [
        '/api/v1/satellites/{satelliteID}/external-sources:',
        'SatelliteSourceIntelligence:',
    ],
    "docs/satellite-real-sources.md": [
        'external_community',
        'radio_activity_context_only',
    ],
}

errors = []
for relative, fragments in checks.items():
    path = root / relative
    if not path.is_file():
        errors.append(f"missing {relative}")
        continue
    text = path.read_text(encoding="utf-8")
    for fragment in fragments:
        if fragment not in text:
            errors.append(f"{relative}: missing {fragment!r}")

client = (root / "apps/backend/internal/satnogs/client.go").read_text(encoding="utf-8")
if '"satellite__norad_cat_id": strconv.Itoa(noradID)' in client.split('observationURL :=', 1)[-1]:
    errors.append("SatNOGS Network observation filter must use norad_cat_id")

if errors:
    print("satellite real sources contract: ERROR", file=sys.stderr)
    for error in errors:
        print(f"- {error}", file=sys.stderr)
    raise SystemExit(1)
print("satellite real sources contract: OK")
