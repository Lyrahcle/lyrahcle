# Raspberry Pi plug-and-play station

Block 4 provides a reproducible Raspberry Pi OS Lite 64-bit station profile. It deliberately does not ship an opaque prebuilt image: the repository builds the ARM64 agent and applies reviewed files to an official Raspberry Pi OS root filesystem. A physical-device acceptance run is still required before calling the hardware production-ready.

## Included workflow

1. `deploy/raspberry-pi/build-arm64.sh` cross-compiles a static Linux ARM64 agent and records its SHA-256.
2. `deploy/raspberry-pi/install.sh` installs the binary, a non-login `lyrahcle` service account, hardened systemd units and operator tools on an already booted Pi.
3. `deploy/raspberry-pi/prepare-rootfs.sh` applies the same payload to a mounted Raspberry Pi OS Lite root filesystem for a reproducible custom-image pipeline.
4. `sudo lyrahcle-enroll` validates HTTPS and local trust material, writes a mode `0600` environment file, starts the service, waits for its mTLS identity, removes the one-use token and restarts without that token.
5. `sudo lyrahcle-doctor` produces a readable checklist; `sudo lyrahcle-doctor --json` produces demo/acceptance evidence.

Before enrollment, provision these trust anchors out of band:

- `/etc/lyrahcle/backend-ca.pem`: CA that verifies the backend TLS certificate;
- one or more backend configuration-signing public keys as `/etc/lyrahcle/config-keys/<kid>.pub`;
- `/etc/lyrahcle/update-signing-key.pem`: dedicated Ed25519 public key for agent release bundles.

Do not put private signing keys or an operator JWT on the Pi.

## Secure updates

Place exactly three regular files in `/var/lib/lyrahcle/updates/pending`: `sdr-agent`, `sdr-agent.sha256`, and `sdr-agent.sig`. The timer invokes `lyrahcle-secure-update`, which rejects symlinks, verifies SHA-256 and the Ed25519 signature, installs atomically, restarts the service and restores the previous binary if systemd does not report it active. It does not download arbitrary URLs.

Example release signing on the protected release workstation:

```bash
sha256sum sdr-agent > sdr-agent.sha256
openssl pkeyutl -sign -inkey update-signing-key.pem -rawin -in sdr-agent -out sdr-agent.sig
```

## Configuration for a real pass

Run `scripts/schedule-agent-pass.py` on an operator workstation. It requests the next qualifying AOS/LOS window from Lyrahcle and prints a dry-run plan by default. With `--apply`, it waits until the configured lead time and publishes the frequency, interval, and satellite UUID through the protected agent configuration endpoint. The backend signs the version; the Pi verifies it using its configured public keys.

```bash
python3 scripts/schedule-agent-pass.py \
  --backend https://lyrahcle.example.com \
  --agent-id station-madrid-01 --satellite-id 00000000-0000-0000-0000-000000000000 \
  --latitude 40.4168 --longitude -3.7038 --altitude-m 667 \
  --frequency-hz 437500000 --token-stdin
```

The prediction is geometric and does not account for buildings, antenna pattern, local interference or receiver sensitivity. Review the dry run and TLE freshness before `--apply`.

## Hardware acceptance checklist

Once the device arrives, record Raspberry Pi model/revision, SDR USB identifiers and firmware; run a 24-hour clock/network soak; verify RF frequency accuracy against a known source; test reconnect after power/network loss; test certificate rotation; stage a signed upgrade and a deliberately invalid one; confirm rollback; compare CPU temperature, throttling and dropped samples under sustained capture. Keep `ORBITAL_AUTO_ESCALATION_ENABLED` disabled until those results are representative.

