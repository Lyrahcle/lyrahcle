#!/usr/bin/env python3
from __future__ import annotations

import sys
from pathlib import Path


def require(text: str, needle: str, label: str, failures: list[str]) -> None:
    if needle not in text:
        failures.append(f"{label}: missing {needle!r}")


def main(argv: list[str] | None = None) -> int:
    root = Path((argv or sys.argv[1:])[0] if (argv or sys.argv[1:]) else ".").resolve()
    frontend = root / "apps" / "frontend" / "src"

    failures: list[str] = []
    focus_trap = (frontend / "lib" / "useFocusTrap.ts").read_text(encoding="utf-8")
    accessibility_test = (frontend / "components" / "accessibility.test.jsx").read_text(encoding="utf-8")
    skip_link = (frontend / "components" / "SkipLink.tsx").read_text(encoding="utf-8")

    require(focus_trap, "previouslyFocused", "focus restoration", failures)
    require(focus_trap, "event.key === 'Escape'", "Escape handling", failures)
    require(focus_trap, "event.key !== 'Tab'", "Tab trapping", failures)
    require(focus_trap, "closest('[inert]')", "inert exclusion", failures)
    require(accessibility_test, "axe(baseElement)", "portal-aware axe audit", failures)
    require(accessibility_test, "required password change cannot be dismissed with Escape", "mandatory-dialog Escape test", failures)
    require(accessibility_test, "ConfirmDialog keyboard contract", "alertdialog keyboard test", failures)
    require(skip_link, "#main-content", "skip navigation target", failures)

    modal_files = [
        "ChangePasswordModal.jsx",
        "TwoFactorModal.jsx",
        "SessionsModal.jsx",
        "ConfirmDialog.jsx",
    ]
    for name in modal_files:
        text = (frontend / "components" / name).read_text(encoding="utf-8")
        require(text, "aria-modal=\"true\"", f"{name} modal semantics", failures)
        require(text, "useFocusTrap", f"{name} focus trap", failures)

    if failures:
        for failure in failures:
            print(f"accessibility contract: ERROR: {failure}", file=sys.stderr)
        return 1

    print("accessibility contract: OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
