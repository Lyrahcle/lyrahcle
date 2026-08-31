# Edge node integrity / tamper detection

Lyrahcle SDR agents can measure a deterministic SHA-256 manifest of critical files and report only hashes/metadata to the backend. The backend establishes the first-seen baseline and classifies later reports as `healthy` or `changed`; agents cannot submit the baseline or status themselves.

Set `AGENT_INTEGRITY_PATHS` to a comma-separated allowlist of critical files (for example the deployed agent config, systemd unit, SDR rules, and verification-key files). The agent executable is always included. Existing files from `/etc/os-release` and `/proc/cmdline` are measured by default when no list is supplied.

The report also includes a SHA-256 hash of `/etc/machine-id`, boot ID, and measured file count. File contents, machine-id contents, and private keys are never uploaded.

## Security boundary

This is runtime tamper evidence, not secure boot or hardware attestation. First-seen baseline is trust-on-first-use. For hardened Raspberry Pi deployments, pin provisioning before field deployment and combine this control with read-only root filesystem, verified boot where supported, TPM/secure-element backed identity, physical seals, and periodic re-enrollment/rotation procedures. A future TPM/IMA attestation provider can replace the measurement source without changing the backend/UI contract.
