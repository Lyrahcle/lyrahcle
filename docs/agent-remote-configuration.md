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
```

Distribute only the public key to each enrolled SDR agent:

```bash
AGENT_CONFIG_VERIFY_KEY=/etc/lyrahcle/agent-config-verify.pub
AGENT_CONFIG_POLL_INTERVAL=30s
```

The signing key must be protected independently from the SDR Agent CA key. Compromise of one key must not silently grant both certificate issuance and configuration signing.

## Trust flow

1. An administrator publishes settings for an `active` agent.
2. The backend creates a monotonic immutable version.
3. The backend signs the exact JSON payload with Ed25519 and stores its SHA-256 checksum.
4. The agent polls over its authenticated mTLS channel.
5. The agent verifies checksum, signature, `agent_id`, version, and bounds before applying anything.
6. The agent acknowledges `applied` or `rejected`.
7. The acknowledgement and administrative action are retained for audit.

A presented but invalid signature is never applied. A configured remote-management agent cannot use legacy API-key authentication for this channel because both config endpoints require a valid active client certificate.

## Rollback

Rollback does not modify or reactivate an old database row. The backend copies the selected historical settings into a new monotonic version and signs that new payload. This preserves an append-only operational history.

## Runtime behavior

A verified configuration is applied without restarting the SDR process. The capture loop wakes when a new version arrives and uses the new frequency and interval on subsequent readings. Certificate rotation and remote configuration remain separate maintenance loops.

## Operational guidance

Use small staged changes, confirm `applied` acknowledgements, and quarantine an agent whose behavior does not match expected configuration. Future fleet-policy work should add hardware capability constraints and staged rollout rather than widening this endpoint into a generic command channel.
