# IQ Forensics

The SDR agent can record raw IQ samples and replay a saved capture through the normal RF ingestion path. Recording and replay are opt-in.

## Record

Set:

```bash
IQ_CAPTURE_DIR=/var/lib/lyrahcle/captures
IQ_CAPTURE_INTERVAL_SECONDS=60
IQ_CAPTURE_SAMPLES=8192
```

Each capture produces:

- `<agent>_<timestamp>.cf32`: interleaved little-endian float32 I/Q samples.
- `<agent>_<timestamp>.json`: capture metadata.

The default device is currently simulated. Captures therefore validate the pipeline and forensic workflow, not real RF hardware fidelity.

## Replay

Run the agent with a saved metadata file:

```bash
REPLAY_ROOT=/captures
REPLAY_FILE=sdr-agent_1735300000000000000.json
REPLAY_WINDOW_SAMPLES=65536
REPLAY_DELAY_MS=300
```

Replay paths are confined to `REPLAY_ROOT`: absolute `REPLAY_FILE` values require an explicit root, traversal (`../`) is rejected, and capture metadata/sample symlinks are not followed.

Replay converts consecutive IQ windows into relative power readings and posts them using reconstructed historical timestamps. This reuses the backend's normal anomaly-detection path.

Power values are relative estimates based on mean sample energy. They are useful within a capture, but are not calibrated dBm measurements across different devices, gains, or sessions.

Run replay as a one-shot container rather than setting `REPLAY_FILE` permanently on a restarting Compose service:

```bash
docker compose run --rm -e REPLAY_ROOT=/captures -e REPLAY_FILE=example.json sdr-agent
```

## Evidence handling

For investigative use, preserve the original `.cf32` and `.json` files together, record their cryptographic hashes externally, avoid modifying the originals, and work from copies. Lyrahcle does not currently implement evidence signing or chain-of-custody management.
