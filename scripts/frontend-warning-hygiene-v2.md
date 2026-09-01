# Frontend warning hygiene v2

This one-time migration helper targets known React `act(...)` warnings in:

- `FleetRolloutPanel.test.jsx`
- the `SettingsPanel` cases inside `accessibility.test.jsx`

It adds a small `flushReactUpdates()` helper to the test files and awaits it at the end of the exact warning-producing tests. The migration is fail-closed and idempotent: both target files must match the expected test names before either file is written.

After the migration is stable, the one-time apply/test scripts can be removed in housekeeping; the focused warning gate remains useful until the global Vitest console-warning gate replaces it.
