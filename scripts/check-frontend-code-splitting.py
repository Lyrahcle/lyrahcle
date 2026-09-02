#!/usr/bin/env python3

from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
DASHBOARD = ROOT / "apps/frontend/src/components/Dashboard.tsx"
HELPER = ROOT / "apps/frontend/src/lib/lazyComponent.tsx"

MIN_LAZY_DECLARATIONS = 10

LAZY_IMPORT_RE = re.compile(
    r"""^import\s*\{\s*lazyComponent\s*\}\s*from\s*['"]\.\./lib/lazyComponent['"]\s*;?\s*$""",
    re.MULTILINE,
)

LAZY_DECL_RE = re.compile(
    r"""^\s*const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*lazyComponent(?:<[^>\n]+>)?\(\s*\(\s*\)\s*=>\s*import\(\s*['"]([^'"]+)['"]\s*\)\s*\)\s*;?\s*$""",
    re.MULTILINE,
)


def fail(message: str) -> int:
    print(f"frontend code-splitting contract: ERROR: {message}", file=sys.stderr)
    return 1


def validate_no_lazy_const_inside_import(text: str) -> str | None:
    in_multiline_import = False

    for lineno, line in enumerate(text.splitlines(), start=1):
        stripped = line.strip()

        if not in_multiline_import and stripped.startswith("import "):
            if "{" in stripped and "} from " not in stripped:
                in_multiline_import = True

        elif in_multiline_import:
            if stripped.startswith("const "):
                return (
                    f"lazy declaration found inside import block "
                    f"at Dashboard.tsx:{lineno}"
                )

            if "} from " in stripped:
                in_multiline_import = False

    if in_multiline_import:
        return "unterminated multiline import in Dashboard.tsx"

    return None


def main() -> int:
    if not DASHBOARD.is_file():
        return fail(f"missing {DASHBOARD.relative_to(ROOT)}")

    if not HELPER.is_file():
        return fail(f"missing {HELPER.relative_to(ROOT)}")

    dashboard = DASHBOARD.read_text(encoding="utf-8")
    helper = HELPER.read_text(encoding="utf-8")

    if "lazyComponent" not in helper:
        return fail("lazyComponent helper does not define lazyComponent")

    imports = LAZY_IMPORT_RE.findall(dashboard)
    if len(imports) != 1:
        return fail(
            "Dashboard.tsx must import lazyComponent exactly once "
            f"(found {len(imports)})"
        )

    structural_error = validate_no_lazy_const_inside_import(dashboard)
    if structural_error:
        return fail(structural_error)

    declarations = LAZY_DECL_RE.findall(dashboard)

    if len(declarations) < MIN_LAZY_DECLARATIONS:
        return fail(
            f"expected at least {MIN_LAZY_DECLARATIONS} lazy declarations, "
            f"found {len(declarations)}"
        )

    names = [name for name, _ in declarations]

    duplicates = sorted(
        name for name in set(names)
        if names.count(name) > 1
    )
    if duplicates:
        return fail(
            "duplicate lazy declarations: " + ", ".join(duplicates)
        )

    dashboard_dir = DASHBOARD.parent

    missing_modules = []
    for name, module_path in declarations:
        if not module_path.startswith("."):
            continue

        target = (dashboard_dir / module_path).resolve()

        try:
            target.relative_to(ROOT)
        except ValueError:
            return fail(
                f"{name} lazy import escapes repository: {module_path}"
            )

        if not target.is_file():
            missing_modules.append(f"{name} -> {module_path}")

    if missing_modules:
        return fail(
            "missing lazy-import modules: " + ", ".join(missing_modules)
        )

    first_import = dashboard.find(
        "import { lazyComponent } from '../lib/lazyComponent'"
    )
    first_decl_match = re.search(r"lazyComponent(?:<[^>\n]+>)?\(\(\) => import\(", dashboard)
    first_decl = first_decl_match.start() if first_decl_match else -1

    if first_import < 0 or first_decl < 0 or first_import > first_decl:
        return fail(
            "lazyComponent import must appear before lazy declarations"
        )

    print(
        "frontend code-splitting contract: OK "
        f"({len(declarations)} lazy declarations)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
