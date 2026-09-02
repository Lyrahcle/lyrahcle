# Frontend accessibility baseline

Lyrahcle's frontend accessibility baseline targets WCAG 2.2 AA-compatible interaction patterns for the application shell and critical operator workflows.

## Automated baseline

The repository enforces the following patterns:

- A skip link moves keyboard users directly to the main application content.
- Interactive controls expose visible `:focus-visible` state.
- Critical forms use explicit label/control associations.
- Dynamic success and error feedback uses status/alert semantics where appropriate.
- Modal dialogs expose `role="dialog"` or `role="alertdialog"`, `aria-modal="true"`, and an accessible name.
- Modal keyboard focus moves inside on open, cannot escape with Tab/Shift+Tab, and returns to the previously focused trigger on unmount.
- Escape dismissal is centralized through the focus-trap helper and can be disabled for mandatory/busy dialogs.
- Automated `jest-axe` checks include portal-rendered modal content via `baseElement`, not only the React render container.

The contract checker `scripts/check-accessibility-contract.py` protects the critical modal and skip-navigation wiring against accidental removal.

## Release-candidate manual gate

Automated checks do not prove WCAG conformance. Before a v1.0 release candidate is accepted, perform and retain evidence for:

1. Full keyboard-only traversal of login, Mission Control, RF, incidents, satellites, settings, user/RBAC administration, and critical dialogs.
2. Screen-reader smoke testing of navigation landmarks, dialog announcements, form errors, tables, status messages, and incident workflows.
3. 200% browser zoom and narrow-viewport checks without loss of required functionality.
4. Reduced-motion preference verification for animated/3D views.
5. Contrast review for text, focus indicators, status badges, charts, and disabled controls in light and dark themes.

Any failure in the manual gate remains a release blocker even when automated axe checks are green.
