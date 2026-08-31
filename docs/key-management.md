# Signing key management

Lyrahcle keeps signing-key use behind a `KeyProvider` contract. Application code asks the provider for an active signing key and receives only an operation capable of signing plus non-secret metadata (`kid`, algorithm); it does not need the raw private key.

## Local provider

The v1 local provider loads a dedicated PKCS#8 Ed25519 private key from `AGENT_CONFIG_SIGNING_KEY_FILE`. The provider validates the key type, derives a stable KID from SHA-256 of the DER-encoded public key, and keeps the private key encapsulated behind the signing interface.

The KID format is:

```text
ed25519-sha256-<64 lowercase hex characters>
```

KIDs are identifiers, not secrets. They may appear in API responses, audit data, and stored configuration-version metadata.

## Version binding

Every newly signed remote-agent configuration stores its signing KID in `agent_config_versions.signing_kid`. The KID served to an agent therefore belongs to the exact immutable version that was signed; the backend must never replace it with whatever key is active later.

Rows created before KID support retain an empty `signing_kid` and are treated as legacy versions.

## Verification keyring

SDR agents can trust multiple public keys through `AGENT_CONFIG_VERIFY_KEY_DIR`. Each regular `*.pem` file is parsed and indexed by the KID derived from its key material. PEM symlinks are rejected so the configured directory cannot escape its trust boundary. Envelopes with a KID are resolved by exact map lookup; the KID is never interpreted as a filename or path.

`AGENT_CONFIG_VERIFY_KEY` is the explicit legacy/default public key. It is used only when an older envelope has no KID. This preserves backward compatibility without an unsafe "try every key" fallback.

## External KMS readiness

The backend provider interface is storage-agnostic and context-aware. A Vault, cloud KMS, HSM, or PKCS#11 implementation can later provide the same `ActiveSigningKey`, `SigningKey`, and `Sign` operations without changing the agent-config signing call sites.

Encrypted backup/restore and provider-specific integrations remain separate v1 checklist items. Manual staged rotation, grace lifecycle and revocation are implemented by the managed local provider; v13.7 adds prepare-only scheduling, lifecycle health and Prometheus metrics.

## Key-purpose separation and v1 lifecycle target

Remote-configuration signing and JWT signing are separate cryptographic purposes. They may share the lifecycle/provider architecture, but they must not share key material or KIDs. A future provider implementation must namespace key lookup by purpose before it is used for JWT signing.

The v1 lifecycle target is:

- normal rotation is operator-controlled and auditable;
- the new key becomes `ACTIVE` before new signatures are emitted;
- the previous key may remain `GRACE` only for a bounded compatibility window;
- emergency revocation bypasses grace;
- JWT grace must never exceed the maximum accepted access-token lifetime plus bounded clock skew;
- remote-config grace must be explicitly configured around the maximum supported agent-offline/update window rather than inferred from JWT settings;
- rotation/revocation must expose metrics and an audit event;
- private key bytes must never be returned by an HTTP API or rendered in the UI.

The provider/KID boundary, immutable version binding, manual lifecycle endpoints, Signing Keys UI, prepare-only scheduled rotation, lifecycle health and metrics are implemented. External KMS providers and encrypted key backup/restore remain future blocks.

## Managed local lifecycle (v13.6)

Set `AGENT_CONFIG_SIGNING_KEY_DIR` to enable lifecycle management. The directory is a backend-only trust boundary: Lyrahcle stores PKCS#8 Ed25519 private keys with restrictive permissions and a manifest that tracks lifecycle state. The directory itself must not be a symlink.

If `AGENT_CONFIG_SIGNING_KEY_FILE` is also configured the existing key is imported as the initial `ACTIVE` key the first time the managed directory is initialized. If no bootstrap file is supplied, Lyrahcle generates the initial key locally.

Key states are:

- `STAGED`: generated but never used for signing. Its public key must be distributed to SDR agents first.
- `ACTIVE`: the only key used for new remote-config signatures.
- `GRACE`: the previous active key remains resolvable for a bounded compatibility window.
- `REVOKED`: no longer trusted by the backend; a desired config bound only to this KID is not served to an agent.

Rotation is deliberately two-phase:

1. **Prepare rotation** from Platform → Signing Keys. Lyrahcle generates a new key and returns only its public key.
2. Distribute that public key into the trusted `AGENT_CONFIG_VERIFY_KEY_DIR` on every affected SDR agent.
3. **Activate** the staged key. The old active key becomes `GRACE` for `AGENT_CONFIG_SIGNING_GRACE_PERIOD`.
4. When the grace period expires the old key transitions to `REVOKED` automatically. A platform administrator can revoke a staged/grace key earlier.

The active key cannot be revoked directly; activate a replacement first. This prevents an accidental control-plane outage with no signing key available.

The platform API never returns private-key material. Public verification keys are intentionally exportable because agents require them to verify signed configuration.

`AGENT_CONFIG_SIGNING_GRACE_PERIOD` defaults to `168h` and must be between `1h` and `720h`.

## Scheduled rotation policy and lifecycle health (v13.7)

Managed providers persist a rotation policy in the same protected manifest as the key lifecycle metadata. Platform administrators can configure the policy from **Platform → Remote Config Signing Keys**. Policy changes survive backend restarts and are audited as `signing_key.policy_updated`.

The rotation interval is measured from `activated_at`, not from when a key was originally staged. This matters operationally because a public key may be prepared and distributed well before activation. Older manifests without `activated_at` remain compatible and fall back to `created_at` until the next activation.

Automatic rotation is intentionally **prepare-only**:

1. the backend checks managed signing-key health hourly;
2. once the active key reaches the configured rotation interval, Lyrahcle creates one `STAGED` replacement if auto-prepare is enabled;
3. `signing_key.auto_prepared` is written to the audit log;
4. an operator distributes the staged public key to SDR agents;
5. a platform administrator explicitly activates it.

Automation never calls `Activate`. This preserves the two-phase trust-distribution invariant introduced in v13.6. Grace expiry remains automatic because it only removes trust at the already-declared deadline.

Default policy:

- automatic prepare: disabled;
- rotation interval: 90 days;
- staged-key warning: after 24 hours;
- grace-expiry warning: 24 hours before expiry.

Accepted limits are 1–365 days for the rotation interval and 1–720 hours for each warning window.

The backend exports bounded Prometheus metrics for key counts, rotation due state, staged pending state, lifecycle warning codes, and automation outcomes. Prometheus rules alert on no active key, overdue rotation, staged keys awaiting activation, approaching grace expiry, and automation failures.
