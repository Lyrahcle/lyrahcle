# Critical backend security test coverage

Lyrahcle enforces a minimum 80% Go coverage gate for backend packages that sit directly on critical correlation and escalation decisions.

## CI gate

The backend CI job runs one combined coverage profile across:

- `internal/correlation` — explainable RF cross-domain candidate scoring and context-only handling.
- `internal/detection` — deterministic RF rule matching and severity ordering.
- `internal/tlequality` — TLE freshness and fail-closed orbital escalation eligibility.

This first backend gate is evaluated with `scripts/check-go-coverage.sh` and CI fails below 80%. This is deliberately separate from the existing >80% SDR processing gate.

## Correlation safety assertions

Tests explicitly preserve these safety properties:

- unlinked CVE/asset alerts remain context-only and cannot raise confidence or candidate severity;
- active agent posture does not create a posture finding;
- quarantined/revoked posture is surfaced as an explainable critical candidate;
- unsupported detection operators fail closed;
- invalid TLE-quality policy fails closed and cannot authorize orbital escalation;
- correlation confidence remains bounded.

The cross-domain engine remains candidate-only: the correlation summary does not directly overwrite incident severity. Existing domain-specific policy and physical RF validation gates remain authoritative.

Authentication has extensive unit/E2E coverage but remains outside this first measured backend gate until its combined percentage is measured independently; the roadmap item stays open until all selected critical backend paths are gated.
