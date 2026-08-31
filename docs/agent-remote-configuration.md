# Signed Remote Agent Configuration

Lyrahcle v9.1 adds a deliberately narrow remote-management channel for active mTLS SDR agents. It does not provide remote shell access or arbitrary command execution.

## Managed settings

The first configuration schema contains only:

- `frequency_hz`: 100 kHz through 6 GHz.
- `capture_interval_seconds`: 1 through 3600 seconds.

The same bounds are enforced in the API, PostgreSQL, and the SDR agent.

## Signing keys

Generate a dedicated Ed25519 key pair:

```bash
./scripts/generate-agent-config-key.sh ./certs/agent-config
```

Mount the private key only into the backend:

```bash
AGENT_CONFIG_SIGNING_KEY_FILE=/run/secrets/agent-config-signing.key
AGENT_CONFIG_SIGNING_KEY_DIR=/var/lib/lyrahcle/agent-config-signing
AGENT_CONFIG_SIGNING_GRACE_PERIOD=168h
```

Distribute only public keys to enrolled SDR agents. New signed envelopes carry a stable `kid` derived from the Ed25519 public key, so agents can select the correct verification key during rotation:

```bash
AGENT_CONFIG_VERIFY_KEY_DIR=/etc/lyrahcle/agent-config-keys
AGENT_CONFIG_POLL_INTERVAL=30s
```

The directory may contain multiple regular `*.pem` public keys (for example, the current key plus a grace-period key). PEM symlinks are rejected. The filename is not trusted as the identifier; Lyrahcle derives the KID from the public key itself.

For compatibility with signed envelopes created before KID support, keep the old public key explicitly configured while legacy versions may still be delivered:

```bash
AGENT_CONFIG_VERIFY_KEY=/etc/lyrahcle/agent-config-verify-legacy.pub
AGENT_CONFIG_VERIFY_KEY_DIR=/etc/lyrahcle/agent-config-keys
```

Legacy envelopes without `kid` are verified only with `AGENT_CONFIG_VERIFY_KEY`; Lyrahcle does not try every key in the keyring. Once no pre-KID versions remain relevant, the legacy variable can be removed.

The signing key must be protected independently from the SDR Agent CA key. Compromise of one key must not silently grant both certificate issuance and configuration signing.

## Trust flow

1. An administrator publishes settings for an `active` agent.
2. The backend creates a monotonic immutable version.
3. The backend asks a `KeyProvider` for the active signing key, signs the exact JSON payload with Ed25519, and stores the SHA-256 checksum plus that key's stable KID with the immutable version.
4. The agent polls over its authenticated mTLS channel and receives the persisted KID in the signed envelope.
5. The agent resolves the public key by KID, then verifies checksum, signature, `agent_id`, version, and bounds before applying anything.
6. The agent acknowledges `applied` or `rejected`.
7. The acknowledgement and administrative action are retained for audit.

A presented but invalid signature is never applied. A configured remote-management agent cannot use legacy API-key authentication for this channel because both config endpoints require a valid active client certificate.

## Rollback

Rollback does not modify or reactivate an old database row. The backend copies the selected historical settings into a new monotonic version and signs that new payload. This preserves an append-only operational history.

## Runtime behavior

A verified configuration is applied without restarting the SDR process. The capture loop wakes when a new version arrives and uses the new frequency and interval on subsequent readings. Certificate rotation and remote configuration remain separate maintenance loops.

## Operational guidance

Use small staged changes, confirm `applied` acknowledgements, and quarantine an agent whose behavior does not match expected configuration. Future fleet-policy work should add hardware capability constraints and staged rollout rather than widening this endpoint into a generic command channel.


## Orbital target

A signed configuration may include an optional `satellite_id`. The backend accepts only a currently tracked satellite UUID. The SDR agent verifies the signed payload before updating its target and propagates that ID in subsequent RF readings.

The satellite target is metadata for correlation and does not grant command authority over a spacecraft.


For managed rotation, prepare the new key in Platform → Signing Keys, copy only the returned public key into each agent's `AGENT_CONFIG_VERIFY_KEY_DIR`, then activate the staged KID. Do not activate before public-key distribution is complete.

## Fleet staged rollouts (v1.4.1)

Tenant operators with `agents.manage` can create staged fleet rollouts from the Connectivity UI. A rollout stores an explicit active-agent cohort, signed desired configuration, batch size, rejection threshold, and soak target. Promotion is operator-controlled: `start` issues the first canary batch and `advance` is accepted only after the current batch has acknowledged and the soak interval has elapsed. If the cumulative decided targets exceed the configured rejection percentage, promotion fails closed and the rollout enters `failed`.

Operators can pause/resume a running rollout and roll it back from the UI. Rollback restores each target's previous desired version; agents that had no desired configuration before the rollout have their desired pointer cleared while immutable signed history is retained. All rollout routes are tenant scoped and guarded by `agents.read` / `agents.manage`, and the authorization inventory test treats every rollout endpoint as a reviewed sensitive route.

### Fleet policy and acknowledgement deadlines (v14.6.5)

New staged rollouts fail closed around hardware eligibility and liveness. Creation accepts only active targets whose latest SDR capability report is fresh and `compatible`, whose integrity manifest has not changed from its baseline, and whose reported tuning range covers the requested frequency. The same policy is re-evaluated immediately before every batch so a device that becomes stale, incompatible, tampered, or out-of-range cannot receive the next staged configuration. The Connectivity UI exposes each active agent as `eligible` or `blocked` with the policy reason instead of leaving this protection API-only.

Each rollout also stores an `ack_timeout_seconds` deadline (30-86400 seconds, default 900). If any issued target remains unacknowledged past that deadline, the next promotion attempt transitions the rollout to `failed` and records `agent.rollout_failed`; the UI surfaces the countdown and changes the promotion control to `Fail overdue batch` once the deadline expires. This timeout complements, rather than replaces, the rejection threshold and soak period.
