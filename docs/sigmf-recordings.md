# SigMF Recordings

The agent reads and writes [SigMF](https://sigmf.org/) 1.0.0 recordings. This makes archived IQ
portable in both directions: Lyrahcle captures can be analysed by third-party tooling, and public
recordings can be replayed through the same spectral pipeline that processes live radio.

Without SDR hardware this is the practical way to exercise acquisition, Doppler estimation and
detection against real signals.

## Writing

Set `IQ_CAPTURE_SIGMF=true` and the capture loop writes `<id>.sigmf-meta` and `<id>.sigmf-data`
instead of the native `.json` + `.cf32` pair. Recordings are written as `cf32_le`, and the
SHA-512 of the data file is stored in `core:sha512` so tampering is detectable on read.

The Lyrahcle extension namespace records provenance: `lyrahcle:agent_id`, `lyrahcle:capture_id`
and `lyrahcle:evidence_kind`.

## Reading and replaying

`REPLAY_FILE` accepts either format; the extension decides the loader. All existing replay
options are unchanged:

```
REPLAY_FILE=community.sigmf-meta REPLAY_ROOT=/var/lib/lyrahcle/captures \
REPLAY_SPECTRAL=true SDR_FFT_SIZE=4096 ./agent
```

Supported datatypes on read: `cf32_le`, `ci16_le`, `ci8` and `cu8`. `cu8` matters most in
practice, because it is what `rtl_sdr` and most public community recordings produce.

## Provenance

Imported recordings are never presented as agent measurements. The loader labels them:

| Field | Values |
|---|---|
| `evidence_kind` | `agent_measured` when the recording carries a Lyrahcle capture identity, otherwise `external_recording` |
| `timestamp_from` | `metadata` when `core:datetime` is present, otherwise `file_modification_time` |
| `integrity` | `verified` when `core:sha512` matched, otherwise `not_declared` |

A recording without a declared datetime is still usable, but its readings inherit a filesystem
timestamp and must not be treated as timed evidence.

## Validation and safety

Metadata is validated before any sample is decoded: version, supported datatype, finite and
positive sample rate, ordered capture segments starting at sample 0, RFC3339 datetimes, bounded
segment and annotation counts, and rejection of non-conforming `core:dataset` references. Segment
offsets are re-checked against the decoded sample count.

Paths resolve through the existing capture-root guard, so symlinks, absolute paths and traversal
outside `REPLAY_ROOT` are refused. Metadata is capped at 8 MiB and data at 512 MiB. The parser has
a fuzz target (`FuzzParse`) asserting that anything it accepts also passes validation.

## Acceptance test without hardware

`internal/replay` includes an end-to-end test that synthesises a `cu8` recording with a known
carrier offset, loads it through the SigMF path and asserts the estimator recovers the offset
within 50 Hz, then builds a Doppler track from it. Use the same shape with real public recordings
to build the evidence set required by `docs/rf-real-acquisition.md`.
