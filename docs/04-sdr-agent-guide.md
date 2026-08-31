# SDR Agent Guide

This guide covers SDR agent enrollment, authentication, remote configuration, hardware capabilities, and fleet management.

## Overview

Lyrahcle v9.0+ uses certificate-based SDR agent identity with mTLS. v9.1 adds signed remote configuration management. API keys remain available only as a migration path during upgrade.

## Agent Identity and Enrollment (v9.0+)

### Trust Model

- Each SDR agent creates its own ECDSA P-256 private key **locally** (private key never leaves the agent)
- Administrator issues a short-lived, single-use enrollment token
- Agent submits enrollment token + CSR and receives a client certificate from the Lyrahcle SDR Agent CA
- Agent certificates are bound to the agent ID and include SPIFFE identity (`spiffe://lyrahcle/agent/<agent_id>`)
- Certificates are valid **only for TLS client authentication**

### Backend Setup

Generate a dedicated agent CA:

```bash
./scripts/generate-agent-ca.sh ./certs/agents
```

Configure the backend:

```bash
AGENT_CA_CERT_FILE=/run/secrets/agent-ca.crt
AGENT_CA_KEY_FILE=/run/secrets/agent-ca.key
AGENT_CERTIFICATE_TTL=168h          # 7 days
AGENT_ENROLLMENT_TTL=15m             # Single-use token lifetime
```

**Important:** The backend must serve HTTPS. When the agent CA is configured, the Go TLS server requests and validates optional client certificates. Browser clients can connect without a client certificate; agent endpoints explicitly require valid active certificates.

**Security:** The agent CA private key is a high-value secret. Production deployments should mount it read-only from a secrets manager or HSM-backed workflow, not stored in the repository or container image.

### Enrollment Procedure

1. **Administrator creates enrollment token** (from Connectivity UI or API):

   ```bash
   POST /api/v1/agents/enrollment-tokens
   {"agent_id":"sdr-agent-madrid"}
   ```

   The plaintext token is returned **once** and expires automatically after `AGENT_ENROLLMENT_TTL`.

2. **Configure the SDR agent** with the one-time token:

   ```bash
   BACKEND_URL=https://lyrahcle.example.com
   AGENT_ID=sdr-agent-madrid
   AGENT_ENROLLMENT_TOKEN=<one-time-token>
   AGENT_CA_CERT=/etc/lyrahcle/backend-server-ca.crt
   AGENT_CERT_FILE=/var/lib/lyrahcle/identity/agent.crt
   AGENT_KEY_FILE=/var/lib/lyrahcle/identity/agent.key
   AGENT_ROTATE_BEFORE=24h
   ```

3. **Agent startup sequence:**
   - Generates ECDSA P-256 private key (if not present)
   - Creates Certificate Signing Request (CSR)
   - Submits CSR + enrollment token to backend
   - Receives signed client certificate from Agent CA
   - Stores private key with mode `0600` (read-only)
   - Switches to mTLS authentication for all future requests

The enrollment token is **consumed** after one use and cannot be replayed.

### Certificate Lifecycle

#### Automatic Rotation

When the current certificate approaches `AGENT_ROTATE_BEFORE`:

1. Agent creates new private key and CSR
2. Agent calls:
   ```bash
   POST /api/v1/agents/me/certificate/rotate
   ```
3. Request must authenticate with **current** client certificate
4. Old certificate is atomically marked `superseded`
5. New certificate takes effect immediately

#### Quarantine and Revocation

Administrators can set an agent identity to:

- **`active`** — Certificate authentication allowed (default)
- **`quarantined`** — Certificate remains recorded but all mTLS authentication is denied
- **`revoked`** — Active certificate is revoked and identity cannot be reactivated

**Important:** Revocation is **intentionally irreversible**. A replacement physical station should use a new `agent_id` to avoid confusion with previously revoked identities.

#### Recovery from Expired Certificate

If a certificate expires before rotation succeeds:

1. Administrator issues a new enrollment token for the **same** `agent_id`
2. Backend permits this recovery only when no valid active certificate remains
3. Backend supersedes the expired certificate before issuing the replacement

### Migration from API Keys

Existing `X-Agent-Key` authentication remains accepted during transition:

- If a client certificate is configured, the SDR agent uses mTLS and **does not** send the API key
- Dual-authentication is not supported: either mTLS or legacy key, not both
- New deployments should **always** prefer enrollment and mTLS
- Legacy API key support is for migration only and will be removed in a future major version

## Remote Configuration (v9.1+)

### Overview

After enrollment, v9.1 can optionally manage a narrow signed capture policy. The mTLS certificate authenticates the transport; separate Ed25519 keys authenticate each configuration version.

**This does NOT provide:**
- Remote shell access
- Arbitrary command execution
- Privilege escalation
- Software updates (managed separately)

### Managed Settings

The configuration schema contains **only:**

- `frequency_hz` — Tuned frequency (100 kHz to 6 GHz)
- `capture_interval_seconds` — Sampling interval (1 to 3600 seconds)
- `satellite_id` — Optional target satellite UUID (signed payload only)

Bounds are enforced consistently in API, PostgreSQL, and SDR agent.

### Configuration Signing Keys

Generate a dedicated Ed25519 key pair:

```bash
./scripts/generate-agent-config-key.sh ./certs/agent-config
```

**Key distribution:**

- Mount private key **only** into the backend:
  ```bash
  AGENT_CONFIG_SIGNING_KEY_FILE=/run/secrets/agent-config-signing.key
  AGENT_CONFIG_SIGNING_KEY_DIR=/var/lib/lyrahcle/agent-config-signing
  AGENT_CONFIG_SIGNING_GRACE_PERIOD=168h  # Key rotation overlap
  ```

- Distribute **only public keys** to enrolled SDR agents:
  ```bash
  AGENT_CONFIG_VERIFY_KEY_DIR=/etc/lyrahcle/agent-config-keys
  AGENT_CONFIG_POLL_INTERVAL=30s
  ```

The directory may contain multiple regular `*.pem` public keys (current + grace-period). PEM symlinks are rejected. Lyrahcle derives the KID (Key ID) from the public key itself, not the filename.

**Compatibility:** For compatibility with pre-KID envelopes:

```bash
AGENT_CONFIG_VERIFY_KEY=/etc/lyrahcle/agent-config-verify-legacy.pub
```

Once no pre-KID versions remain relevant, this variable can be removed.

**Security:** The signing key must be protected **independently** from the SDR Agent CA key. Compromise of one key must not silently grant both certificate issuance and configuration signing.

### Configuration Trust Flow

1. Administrator publishes settings from the Connectivity UI or API
2. Backend creates a **monotonic immutable version** (no mutations to old versions)
3. Backend signs the exact JSON payload with Ed25519 and stores:
   - SHA-256 checksum
   - Signing key's stable KID
   - Agent ID
   - Version number
4. Agent polls over authenticated mTLS and receives the signed envelope
5. Agent verifies:
   - Public key by KID
   - SHA-256 checksum
   - Ed25519 signature
   - Agent ID matches
   - Version is new
   - Bounds are valid (frequency/interval)
6. If all checks pass, agent applies configuration **without restarting**
7. Agent acknowledges `applied` or `rejected` (retained for audit)

**Validation:** A presented but invalid signature is **never applied**. Agents authenticated with client certificates **cannot** use legacy API-key authentication for config endpoints.

### Rollback

Rollback does **not** modify or reactivate an old database row:

1. Backend copies the selected historical settings into a **new** monotonic version
2. Backend signs that new payload
3. Agent applies the new rollback version

This preserves an **append-only operational history** for audit and forensics.

### Runtime Behavior

- Verified configuration is applied **without restarting** the SDR process
- Capture loop wakes when new version arrives
- New frequency and interval take effect on subsequent readings
- Certificate rotation and remote configuration are **separate** maintenance loops
- Frequency/interval changes are **not** instant but effective within the next poll cycle

### Operational Guidance

1. Use **small staged changes** (test one frequency or interval at a time)
2. **Confirm `applied` acknowledgements** before advancing to next batch
3. **Quarantine agents** whose behavior doesn't match expected configuration
4. Use fleet policies to validate hardware capability before issuing configuration
5. Monitor acknowledgement timeouts and reject rates

## Hardware Capabilities (v13.10+)

### Device Inventory

Lyrahcle agents report SDR hardware inventory through the machine-authenticated endpoint:

```bash
POST /api/v1/agents/me/device
```

The backend derives the tenant from the authenticated agent credential. Callers cannot select another organization.

### Reported Capabilities

The inventory report includes:

- **Basic:** Driver, model, serial number
- **System:** Architecture, OS, driver/firmware versions
- **RF:** Frequency range, sample rates, bandwidths, gain elements, channel count
- **Hardware:** GPS/PPS capability, temperature sensor, sample-drop counters

Reports are stored as JSONB alongside existing device metadata for forward-compatibility with future fleet policies.

### Capability States

The UI labels devices as:

- **`compatible`** — Self-reported software capabilities meet Lyrahcle v1 baseline
  - **Note:** This does NOT mean the hardware has completed the physical RTL-SDR/HackRF lab-validation gate
  - `compatible` is a software self-report only; physical validation is separate

- **`degraded`** — Known SDR drivers with incomplete capability data
  
- **`unsupported`** — Unknown or RX-incapable devices
  
- **`stale`** — Reports older than 15 minutes

### Physical Hardware Validation

Hardware capability validation is a **separate gate** before v1.0:

- RTL-SDR and HackRF physical testing (lab validation)
- Frequency/sample-rate/bandwidth verification
- Gain element behavior and temperature range
- GPS/PPS functionality (if claimed)
- RF performance under expected conditions

This validation is **not yet complete** for all claimed capabilities.

## Fleet Staged Rollouts (v14.6.1+)

### Overview

Tenant operators with `agents.manage` capability can create staged fleet rollouts from the Connectivity UI. Rollouts store:

- Explicit active-agent cohort
- Signed desired configuration
- Batch size
- Rejection threshold
- Soak target (time between batches)

### Promotion Model

1. **`start`** — Issues first canary batch to selected agents
2. **Wait for acknowledgements** — Agents respond `applied` or `rejected`
3. **`advance`** — Promoted only after:
   - Current batch all acknowledged
   - Soak interval elapsed
   - Cumulative rejection rate **below** threshold
   - All targets still eligible per fleet policy
4. **Rollback** — Restore each target's previous desired version
   - Agents that had no desired configuration have pointer cleared
   - Immutable signed history is retained

### Fleet Policy and Eligibility (v14.6.5)

Staged rollouts enforce hardware and liveness policy:

- **Fresh capability reports** — Latest SDR capability report ≤ 15 minutes old
- **Compatible status** — Device status must be `compatible` (not `stale`, `degraded`, `unsupported`)
- **Integrity baseline** — Integrity manifest unchanged from first-seen baseline
- **Frequency coverage** — Reported tuning range covers the requested frequency
- **Re-evaluation** — Same policy checks immediately before each batch promotion

The Connectivity UI displays each target as:

- **`eligible`** — Passes all policy checks
- **`blocked`** — Fails policy check (UI shows reason)

### Acknowledgement Timeouts

Each rollout stores `ack_timeout_seconds` (30–86,400 seconds, default 900 seconds):

- If any issued target remains **unacknowledged** past the deadline
- Next promotion attempt transitions rollout to `failed`
- Audit event recorded: `agent.rollout_failed`
- UI surfaces countdown and changes promotion control once deadline expires

Acknowledgement timeouts **complement** rejection threshold and soak period (not a replacement).

### Authorization

All rollout routes are:

- **Tenant-scoped** — Organization boundaries enforced
- **Guarded by permissions:** `agents.read` / `agents.manage`
- **Audit coverage:** Authorization inventory test treats all rollout endpoints as reviewed sensitive routes

## Edge Integrity Monitoring

Lyrahcle SDR agents can measure a deterministic SHA-256 manifest of critical files and report only hashes/metadata to the backend:

```bash
AGENT_INTEGRITY_PATHS=/etc/systemd/system/lyrahcle-sdr.service,/etc/lyrahcle/sdr-rules.json
```

The backend:
1. Establishes first-seen baseline
2. Classifies later reports as `healthy` or `changed`
3. Agents cannot submit their own baseline or status

### Measurement Scope

- **Always:** Agent executable (SHA-256)
- **Default:** `/etc/os-release`, `/proc/cmdline` (if no allowlist)
- **Optional:** Custom files in `AGENT_INTEGRITY_PATHS`

Report also includes:
- SHA-256 hash of `/etc/machine-id` (not the content)
- Boot ID
- Measured file count

**Not transmitted:** File contents, machine-id contents, or private keys.

### Security Boundary

This is **runtime tamper evidence**, not:
- Secure boot
- Hardware attestation
- TPM-backed identity (future work)

For hardened deployments (e.g., Raspberry Pi):

1. Pin provisioning before field deployment
2. Combine with read-only root filesystem
3. Enable verified boot (if supported)
4. Use TPM/secure-element backed identity (future)
5. Physical seals and periodic re-enrollment
6. Regular integrity checks (automated alerts)

## Checklist for Deployment

### Initial Setup
- [ ] Agent CA generated and mounted read-only
- [ ] Configuration signing keys generated
- [ ] Public keys distributed to agents
- [ ] HTTPS configured on backend
- [ ] `AGENT_ENROLLMENT_TTL` appropriate for your workflow

### Ongoing Operations
- [ ] Enrollment tokens never logged or shared
- [ ] Private keys stored with `0600` permissions
- [ ] Certificate rotation working (check logs)
- [ ] Hardware capability reports fresh (≤15 min old)
- [ ] Integrity baselines established on first report
- [ ] Staged rollouts tested with canary agents first
- [ ] Acknowledgement timeouts monitored
- [ ] Revoked agents cannot reconnect (verify)

### Security Review
- [ ] Agent CA and config signing keys in separate vaults
- [ ] Key rotation plan documented
- [ ] Agent enrollment and certificate rotation automated
- [ ] Fleet policies validated before production rollout
