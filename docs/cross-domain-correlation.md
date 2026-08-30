# Cross-domain correlation

Lyrahcle v9.4 adds an explainable correlation layer to RF detection incidents. The goal is to help an operator understand which independent domains corroborate an incident without converting weak or globally-scoped context into automatic severity escalation.

## Corroborating domains

The v9.4 evaluator can use the following evidence when it is available:

- the originating RF anomaly;
- matched deterministic detection rules;
- orbital candidate evidence produced by v9.2/v9.2.2;
- SDR agent identity posture;
- recent audit activity for the authenticated agent.

The engine emits a bounded confidence value, candidate severity, a list of corroborating domains, and human-readable reasons. This summary is stored as incident evidence with type `correlation` and reference `cross-domain-v9.4`.

## Context-only evidence

Recent unacknowledged asset CVE alerts are useful operator context, but the current schema does not prove that a global asset alert caused or corroborated a particular RF anomaly. v9.4 therefore attaches these alerts as `asset_alert_context` with `context_only: true`.

Context-only evidence:

- does not increase correlation confidence;
- does not raise incident severity;
- must not be presented as causal evidence in the UI.

## Severity safety

v9.4 is candidate-only. It does not overwrite the incident severity selected by deterministic detection rules and existing domain-specific policy. In particular, orbital candidate severity remains subject to TLE freshness and `ORBITAL_AUTO_ESCALATION_ENABLED` gating.

This separation allows Lyrahcle to collect operator feedback and representative production-like evidence before any future correlation-driven severity policy is enabled.

## Audit window

For an RF incident associated with an agent, the backend attaches up to 10 audit events for that agent from the previous 24 hours. These events are intended to explain recent enrollment, certificate, configuration, or other authenticated agent activity; they are not treated as malicious merely because they exist.

## Asset-alert context window

The backend may attach up to 5 unacknowledged asset CVE alerts from the previous 7 days as context. A future data-model revision may permit direct asset/satellite/agent relationships; until then these alerts remain context-only.
