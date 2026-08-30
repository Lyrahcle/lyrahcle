# SDR Agent Enrollment and mTLS

Lyrahcle v9.0 supports certificate-based SDR agent identity. API keys remain available only as a migration path.

## Trust model

Each SDR agent creates its own ECDSA P-256 private key locally. The private key never leaves the agent. An administrator issues a short-lived, single-use enrollment token. The agent submits that token together with a CSR and receives a client certificate signed by the Lyrahcle SDR Agent CA.

Agent certificates are bound to the requested `agent_id`, include the URI identity `spiffe://lyrahcle/agent/<agent_id>`, and are valid only for TLS client authentication.

## Backend setup

Generate a dedicated agent CA:

```bash
./scripts/generate-agent-ca.sh ./certs/agents
```

Configure the backend:

```bash
AGENT_CA_CERT_FILE=/run/secrets/agent-ca.crt
AGENT_CA_KEY_FILE=/run/secrets/agent-ca.key
AGENT_CERTIFICATE_TTL=168h
AGENT_ENROLLMENT_TTL=15m
```

The backend must also serve HTTPS. When the agent CA is configured, the Go TLS server requests and validates optional client certificates. Browser clients can still connect without a client certificate, while agent endpoints explicitly require a valid active certificate or a legacy credential during migration.

The agent CA private key is a high-value secret. Production deployments should mount it read-only from a secrets manager or HSM-backed workflow instead of storing it in the repository or container image.

## Enrollment

As an administrator, create a one-time token from Connectivity or:

```text
POST /api/v1/agents/enrollment-tokens
{"agent_id":"sdr-agent-madrid"}
```

The plaintext token is returned once and expires automatically.

Configure the SDR agent:

```bash
BACKEND_URL=https://lyrahcle.example.com
AGENT_ID=sdr-agent-madrid
AGENT_ENROLLMENT_TOKEN=<one-time-token>
AGENT_CA_CERT=/etc/lyrahcle/backend-server-ca.crt
AGENT_CERT_FILE=/var/lib/lyrahcle/identity/agent.crt
AGENT_KEY_FILE=/var/lib/lyrahcle/identity/agent.key
AGENT_ROTATE_BEFORE=24h
```

On first startup the agent generates its key and CSR, enrolls, stores the private key with mode `0600`, and then authenticates with mTLS.

## Rotation

When the current certificate approaches `AGENT_ROTATE_BEFORE`, the agent creates a new private key and CSR and calls:

```text
POST /api/v1/agents/me/certificate/rotate
```

The request must authenticate with the current client certificate. The old certificate is atomically marked `superseded` when the new certificate is stored.

## Quarantine and revocation

Administrators can set an identity to:

- `active`: certificate authentication allowed.
- `quarantined`: certificate remains recorded but all mTLS agent authentication is denied.
- `revoked`: active certificate is revoked and the identity cannot be reactivated.

Revocation is intentionally irreversible. A replacement physical station should use a new `agent_id` so historical evidence cannot be confused with a previously revoked identity.

## Migration from API keys

Existing `X-Agent-Key` authentication remains accepted during the migration period. If a client certificate is configured, the SDR agent uses mTLS and does not send the API key. New deployments should prefer enrollment and mTLS.

If a certificate expires before rotation succeeds, an administrator can issue a new enrollment token for the same `agent_id`. The backend permits this recovery only when no valid active certificate remains and supersedes the expired certificate before issuing the replacement identity.


## Remote configuration

After enrollment, v9.1 can optionally manage a narrow signed capture policy. The mTLS certificate authenticates the agent transport while a separate Ed25519 key authenticates each configuration version. See `docs/agent-remote-configuration.md`.
