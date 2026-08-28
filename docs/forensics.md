# IQ capture & forensic playback

`sdr-agent` can record raw IQ samples to disk and, separately, replay a
previously saved capture back into the backend as historical RF readings —
useful for re-running anomaly detection against a signal after the fact,
without needing the original device or session.

Both are opt-in and off by default; the agent's normal mode (periodic RF
power readings only) is unaffected.

## Recording captures

Set `IQ_CAPTURE_DIR` to enable a background recording loop, independent of
the regular RF readings sent every `CAPTURE_INTERVAL_SECONDS`:

```bash
IQ_CAPTURE_DIR=/var/lib/lyrahcle/captures
IQ_CAPTURE_INTERVAL_SECONDS=60   # default: record every 60s
IQ_CAPTURE_SAMPLES=8192          # default: 8192 samples per capture
```

Each capture writes two files sharing a basename under `IQ_CAPTURE_DIR`:

- `<agent>_<timestamp>.cf32` — raw interleaved little-endian float32 I/Q
  samples, no header (the same convention GNU Radio uses for `.cf32`, so
  captures are importable into other SDR tooling)
- `<agent>_<timestamp>.json` — metadata: agent ID, frequency, sample rate,
  capture time, sample count

Recording currently runs against the mock device (`internal/soapy`'s
`MockDevice`) — real RTL-SDR/HackRF hardware needs a physical device
connected to the agent's host, which this environment doesn't have. The
`soapy.Device` interface and `streaming.StreamManager` used here are
hardware-agnostic; wiring in a real SoapySDR driver is a drop-in swap of
`soapy.NewMockDevice()` for a real implementation in
`cmd/agent/main.go`'s `runIQCaptureLoop`, not a rewrite.

## Replaying a capture

Set `REPLAY_FILE` to a capture's `.json` metadata path to run the agent in
one-shot replay mode instead of the live loop:

```bash
REPLAY_FILE=/var/lib/lyrahcle/captures/sdr-agent-madrid_1735300000000000000.json
REPLAY_WINDOW_SAMPLES=65536   # default: samples per re-ingested reading
REPLAY_DELAY_MS=300           # default: delay between posted readings
```

The agent loads the capture, splits its samples into consecutive windows of
`REPLAY_WINDOW_SAMPLES`, reduces each window to one RF power reading, and
posts them to `POST /api/v1/rf/ingest` with `captured_at` timestamps
reconstructed from the capture's original timeline (not "now") — so the
backend's per-agent anomaly baseline and the dashboard's history reflect
when the signal was actually recorded, and any outlier in the replayed data
is flagged the same way a live one would be.

**The power figure is a relative estimate**, not a calibrated measurement:
it's `10*log10(mean|sample|²)` over each window, useful for spotting
outliers within one replay but not for comparing absolute dBm across
different hardware, gain settings, or capture sessions. Calibrating against
a real device's actual gain chain is tracked as future work alongside real
SDR hardware integration.

After replay, the agent exits — it doesn't fall back to the live loop.

**Don't set `REPLAY_FILE` on the persistent `sdr-agent` Compose service** —
it restarts automatically (`restart: unless-stopped`), so the agent would
exit after replaying and immediately replay the same capture again in a
loop. Run replay as a one-off instead:

```bash
docker compose run --rm -e REPLAY_FILE=/captures/sdr-agent-madrid_....json sdr-agent
```

## Reference

| Variable | Default | Description |
|---|---|---|
| `IQ_CAPTURE_DIR` | unset (disabled) | Directory to record periodic IQ captures to |
| `IQ_CAPTURE_INTERVAL_SECONDS` | `60` | Seconds between recordings |
| `IQ_CAPTURE_SAMPLES` | `8192` | Samples per recording |
| `REPLAY_FILE` | unset (disabled) | Path to a capture's `.json` metadata file; when set, runs replay-only and exits |
| `REPLAY_WINDOW_SAMPLES` | `65536` | Samples per re-ingested power reading |
| `REPLAY_DELAY_MS` | `300` | Delay between posted readings, to stay under the backend's rate limit |

Implementation: `apps/sdr-agent/internal/forensics` (save/load/list
captures) and `apps/sdr-agent/internal/replay` (capture → power readings).
