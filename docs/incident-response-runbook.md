# Lyrahcle Incident Response Runbook

This runbook covers security incidents affecting Lyrahcle itself: compromised credentials or sessions, signing-key compromise, SDR-agent compromise/tamper, suspected tenant-data exposure, and backend/infrastructure compromise.

It complements `SECURITY.md` (vulnerability reporting), `docs/forensics.md` (IQ evidence handling), `docs/key-management.md`, `docs/agent-enrollment.md`, and `docs/backup-and-restore.md`.

## Operating principles

1. **Preserve evidence before destructive remediation when safe.** Record request IDs, timestamps, affected users/agents/organizations, relevant audit events, configuration versions, and hashes of exported evidence.
2. **Contain with the narrowest effective control.** Prefer revoking one session, quarantining one agent, or disabling one tenant before broader shutdowns when the blast radius is known.
3. **Do not trust a compromised component to attest to its own integrity.** Edge integrity state and remote-config signing state are evaluated by the backend/custody boundary.
4. **Rotate after containment.** Rotating credentials before blocking an active attacker can simply disclose the replacement secret.
5. **Keep a written incident timeline.** Use UTC timestamps and preserve the Lyrahcle request ID for every relevant HTTP action.

## Severity and declaration

Declare a Lyrahcle security incident when any of the following is credible:

- unauthorized access to a privileged user or platform-admin account;
- unauthorized use or disclosure of the remote-config signing private key, SDR Agent CA, JWT signing secret, database/Redis credentials, or secret-provider token;
- edge integrity state changes without an approved maintenance explanation;
- a tenant can access another organization's data or a PostgreSQL RLS policy is bypassed unexpectedly;
- an agent identity is cloned, stolen, or used from an unexpected host;
- production data, forensic captures, credentials, or audit evidence are exfiltrated or altered;
- a software/supply-chain compromise affects a deployed Lyrahcle artifact.

Use these operational severities:

- **SEV-1:** confirmed or highly likely cross-tenant exposure, signing/CA compromise, active platform-admin takeover, destructive compromise, or widespread agent impersonation.
- **SEV-2:** confirmed single-tenant compromise, single privileged account compromise, single edge-node compromise, or security-control failure with contained blast radius.
- **SEV-3:** suspicious activity requiring investigation but with no confirmed unauthorized impact.

## Universal first 15 minutes

- Record incident start time in UTC and the reporter/source.
- Capture the current request IDs, affected identities, organization IDs, agent IDs, signing KIDs, and config versions.
- Preserve relevant audit-log entries and backend logs before rotation/restart where practical.
- Confirm whether the incident is still active.
- Reduce blast radius with the specific containment procedure below.
- Assign one incident commander and one evidence owner; avoid conflicting remediation actions.

## Playbook A — compromised user or platform administrator

### Contain

- Revoke the affected user's sessions from the existing user/session controls.
- Disable the account if compromise is confirmed or recovery ownership is uncertain.
- For platform administrators, review the Platform administrators list and remove unauthorized global access immediately.
- Review recent role/RBAC and organization-management audit events.

### Eradicate

- Reset the account password through the approved recovery process.
- Re-enroll TOTP/WebAuthn factors if their trust is in doubt.
- Rotate any secrets the account could retrieve outside Lyrahcle.

### Recover

- Re-enable only after the identity owner is verified and factors are re-established.
- Review all mutations performed during the compromise window.

## Playbook B — remote-config signing key compromise

Treat disclosure or unauthorized signing capability as **SEV-1**.

### Contain

- Pause active Fleet Rollouts.
- Prevent new remote-configuration publication until custody is restored.
- In the external custody system, disable/revoke the compromised KID where supported.
- Preserve the list of signed config versions, KIDs, target agents, and acknowledgements from the suspected window.

### Eradicate

- Stage a new signing key/KID through Platform → Remote Config Signing Keys or the configured external custody system.
- Distribute the new public verification key to agents through the approved trust channel.
- Activate the replacement only after verification-key distribution is confirmed.
- Minimize or eliminate the previous-key grace period when compromise is confirmed.

### Recover

- Reissue known-good desired configurations signed by the new KID.
- Verify acknowledgements and edge integrity before resuming staged rollouts.

## Playbook C — SDR Agent CA / agent identity compromise

### Single agent

- Quarantine the mTLS identity from Connectivity.
- Preserve certificate fingerprint, version, last-seen time, station coordinates, edge-integrity hashes, and recent RF/config activity.
- If theft/clone is confirmed, revoke the identity and enroll a replacement rather than unquarantining it.

### SDR Agent CA compromise

Treat as **SEV-1**.

- Stop issuing/rotating agent certificates from the compromised CA.
- Prepare a replacement CA through the deployment secret-management workflow.
- Inventory all currently trusted agent certificates and plan forced re-enrollment.
- Do not reuse the compromised CA key or extend its lifetime for convenience.

## Playbook D — edge tamper / physical compromise

When Connectivity → Hardware shows `changed` unexpectedly:

- Quarantine the agent before accepting new remote configuration or RF evidence as trusted.
- Compare the measured manifest hash, machine identity hash, boot ID, binary version, and maintenance records.
- Preserve the device filesystem/image or relevant configuration files according to organizational forensic policy before reinstalling.
- If compromise cannot be disproven, rebuild from trusted media and re-enroll with a new agent identity.
- Establish a new integrity baseline only after ownership and software provenance are verified.

A matching TOFU baseline is not hardware attestation. For higher assurance, pair this control with Secure Boot/TPM/IMA in the deployment profile.

## Playbook E — suspected cross-tenant data exposure

Treat credible cross-tenant access as **SEV-1**.

### Contain

- Preserve the exact endpoint, request ID, actor, organization ID, SQL/RLS error context, and returned object identifiers.
- Disable or isolate the affected route/workload if exploitation is ongoing.
- Do **not** globally disable RLS as a diagnostic shortcut.
- If one organization is compromised, disable that organization before broader shutdown where sufficient.

### Investigate

- Reproduce with a dedicated test identity in staging when possible.
- Verify application scoping and PostgreSQL `FORCE ROW LEVEL SECURITY` policy behavior separately.
- Review any control-plane DB scope usage and confirm platform-admin authorization preceded it.
- Search audit events for the affected resource identifiers and actor.

### Recover

- Add a regression test reproducing the boundary failure before restoring normal traffic.
- Determine notification/disclosure obligations with legal/compliance stakeholders before external communication.

## Playbook F — JWT / backend secret compromise

- Revoke active sessions where supported and rotate `JWT_SECRET`; use `JWT_SECRET_PREVIOUS` only for a deliberately bounded overlap when compromise has **not** exposed the previous key.
- For confirmed JWT signing-secret compromise, prefer forced reauthentication rather than a long overlap window.
- Rotate database, Redis, SMTP, signer-service, and other affected credentials in their custody system.
- Restart workloads only after replacement references are available and validate Secret Provider health in Platform.
- Review logs for unauthorized secret-dependent actions during the exposure window.

## Playbook G — database / Redis compromise

- Restrict network access first; preserve snapshots/logs before destructive recovery if safe.
- Rotate database/Redis credentials through the secret provider.
- Validate PostgreSQL RLS policies and privileged roles before reopening traffic.
- Restore from a verified backup if integrity is uncertain; follow `docs/backup-and-restore.md`.
- Invalidate sessions/tokens whose revocation or state may have been tampered with.

## Evidence checklist

Preserve, where relevant:

- Lyrahcle request IDs and UTC timestamps;
- audit-log events;
- user/role/platform-admin state;
- organization IDs and enabled/disabled state;
- agent ID, certificate fingerprint/version, integrity manifest hash, boot ID, hardware metadata;
- remote-config version, KID, payload hash, target list, acknowledgements, rollout state;
- IQ capture `.cf32` + metadata `.json` hashes;
- backend/container/infrastructure logs;
- database backup/snapshot identifiers and hashes/manifests;
- software image digest, SBOM/provenance and deployed commit/release identifier.

Never place private keys, passwords, bearer tokens, recovery codes, or raw secret values into the incident timeline or ticket.

## Recovery gates

Do not close the incident until all applicable gates are satisfied:

- containment is verified and the attacker path is no longer active;
- compromised credentials/keys/identities are revoked or rotated;
- affected systems are rebuilt/restored from a trusted state when integrity is uncertain;
- cross-tenant and authorization boundaries have regression tests where relevant;
- monitoring/alerts are in place for recurrence;
- the incident timeline and evidence inventory are complete;
- disclosure/notification decisions are recorded;
- follow-up fixes have owners and target milestones.

## Post-incident review

Within the organization's chosen review window, document root cause, blast radius, detection source, containment time, recovery time, control failures, what worked, what did not, and concrete preventive actions. Update this runbook whenever an incident exposes an ambiguous or missing operational step.
