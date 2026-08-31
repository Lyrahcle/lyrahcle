#!/usr/bin/env python3
"""Fail CI on high-risk frontend DOM/XSS patterns without an explicit review."""
from __future__ import annotations
import pathlib
import re
import sys

ROOT = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
FRONTEND = ROOT / "apps" / "frontend" / "src"
EXTS = {".js", ".jsx", ".ts", ".tsx"}

# WorldMap is the sole reviewed raw-HTML sink. Its HTML is produced locally by
# dotted-map from bundled static world-map-data.json, not from API/user content.
RAW_HTML_ALLOWLIST = {"components/WorldMap.jsx"}

errors: list[str] = []
for path in FRONTEND.rglob("*"):
    if path.suffix not in EXTS or path.name.endswith((".test.js", ".test.jsx", ".test.ts", ".test.tsx")):
        continue
    rel = path.relative_to(FRONTEND).as_posix()
    text = path.read_text(encoding="utf-8")

    for pattern, label in [
        (r"\binnerHTML\s*=", "innerHTML assignment"),
        (r"\bouterHTML\s*=", "outerHTML assignment"),
        (r"document\.write\s*\(", "document.write"),
    ]:
        if re.search(pattern, text):
            errors.append(f"{rel}: forbidden {label}")

    if "dangerouslySetInnerHTML" in text:
        if rel not in RAW_HTML_ALLOWLIST:
            errors.append(f"{rel}: unreviewed dangerouslySetInnerHTML")
        elif "dangerouslySetInnerHTML={{ __html: baseSvg }}" not in text or "world-map-data.json" not in text:
            errors.append(f"{rel}: reviewed WorldMap raw HTML contract changed")

    # Every new-tab link must explicitly sever opener and referrer relationships.
    for match in re.finditer(r"<a\b[^>]*target=[\"']_blank[\"'][^>]*>", text, re.S):
        tag = match.group(0)
        line = text.count("\n", 0, match.start()) + 1
        rel_matches = re.findall(r"\brel=[\"']([^\"']*)[\"']", tag)
        if len(rel_matches) > 1:
            # A duplicate JSX attribute is valid JS (later key wins in the
            # transpiled object literal), so only the LAST rel value is
            # actually applied at runtime -- any earlier one is silently
            # discarded. This has previously masked a missing `noopener`.
            errors.append(
                f"{rel}:{line}: target=_blank has {len(rel_matches)} rel attributes; "
                "only the last is applied at runtime, so this is always a bug"
            )
            continue
        tokens = set(rel_matches[0].split()) if rel_matches else set()
        missing = {"noopener", "noreferrer"} - tokens
        if missing:
            errors.append(f"{rel}:{line}: target=_blank missing rel tokens: {', '.join(sorted(missing))}")

if errors:
    print("frontend XSS contract: FAILED", file=sys.stderr)
    for error in errors:
        print(f" - {error}", file=sys.stderr)
    raise SystemExit(1)
print("frontend XSS contract: OK")
