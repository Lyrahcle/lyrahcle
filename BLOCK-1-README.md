# Block 1 — Real RF acquisition and spectral analysis

This package replaces the mock-only SDR stack with a real acquisition path and a real analysis
layer. Everything compiles and passes with `CGO_ENABLED=0`, and no new module dependency is
introduced: `apps/sdr-agent/go.mod` still has an empty `require` block.

## Files in this package

### New

| Path | Purpose |
| --- | --- |
| `apps/sdr-agent/internal/dsp/fft.go` | Iterative radix-2 FFT/IFFT, bit reversal, spectrum shift, complex64/128 conversion |
| `apps/sdr-agent/internal/dsp/window.go` | Rectangular, Hann, Hamming and Blackman-Harris windows with coherent gain and equivalent noise bandwidth |
| `apps/sdr-agent/internal/dsp/psd.go` | Welch periodogram, median noise floor, interpolated peak, band power, 99% occupied bandwidth, waterfall rows |
| `apps/sdr-agent/internal/dsp/doppler.go` | Two-stage carrier estimation, complex mixdown, Kay estimator, range-rate conversion, prediction residual |
| `apps/sdr-agent/internal/soapy/rtltcp.go` | RTL-SDR driver over the `rtl_tcp` wire protocol |
| `apps/sdr-agent/internal/soapy/factory.go` | Device selection from a URI (`mock://`, `rtltcp://host:port`) |
| `apps/sdr-agent/internal/capture/spectral.go` | `Source` backed by a real device plus the DSP layer, with reconnection and calibration |
| `apps/sdr-agent/internal/replay/spectral.go` | Offline spectral analysis of archived `.cf32` captures and Doppler track extraction |
| `docs/rf-real-acquisition.md` | Deployment, calibration and the physical validation procedure |
| Five `_test.go` files | 61 test and fuzz functions across the four RF packages |

### Modified

| Path | Change |
| --- | --- |
| `apps/sdr-agent/internal/capture/capture.go` | `Reading` gained an optional `Spectral *SpectralMetrics`; `MockSource` is unchanged |
| `apps/sdr-agent/cmd/agent/main.go` | Device selection from environment, spectral source wiring, richer log line, `REPLAY_SPECTRAL` path, IQ capture loop uses the configured device |
| `.github/workflows/ci.yml` | New 80% coverage gate over `dsp`, `capture` and `soapy`; new fuzz step for the carrier estimator |
| `docs/README.md` | Index entry for the new document |
| `CHANGELOG.md` | Unreleased entry |

## Applying it

The archive mirrors the repository layout, so from the repository root:

```
cp -r <extracted>/apps/sdr-agent/internal/dsp apps/sdr-agent/internal/
cp <extracted>/apps/sdr-agent/internal/soapy/{rtltcp.go,rtltcp_test.go,factory.go,device_test.go} apps/sdr-agent/internal/soapy/
cp <extracted>/apps/sdr-agent/internal/capture/*.go apps/sdr-agent/internal/capture/
cp <extracted>/apps/sdr-agent/internal/replay/spectral*.go apps/sdr-agent/internal/replay/
cp <extracted>/apps/sdr-agent/cmd/agent/main.go apps/sdr-agent/cmd/agent/
cp <extracted>/docs/rf-real-acquisition.md docs/
```

Then review the diffs on `.github/workflows/ci.yml`, `docs/README.md` and `CHANGELOG.md` and merge
them by hand, since those files change often.

## Verifying

```
cd apps/sdr-agent
go build ./...
go vet ./...
go test ./...
go test -race ./...
go test ./internal/dsp ./internal/capture ./internal/soapy -cover
go test ./internal/dsp -run=^$ -fuzz=FuzzEstimateCarrier -fuzztime=30s
```

Measured coverage: `dsp` 82.0%, `capture` 80.8%, `soapy` 84.4%. All packages pass the race detector.

The tests were run against Go 1.23; the module declares Go 1.26.0 as in the repository. If you
build locally with an older toolchain, temporarily lower the `go` directive rather than editing the
source.

## What this does not do

- The `/api/v1/rf/ingest` payload is unchanged, because the backend uses
  `decodeStrictJSON` with `DisallowUnknownFields` and would reject additional fields. Sending the
  spectral metrics upstream needs a coordinated backend change: extend the ingest struct, add a
  migration for the new columns, and only then extend `uplink.Send`.
- Only the RTL-SDR family is supported. HackRF and USRP need either their own `Device`
  implementation or a genuine SoapySDR binding, which would reintroduce cgo and break the `scratch`
  image.
- Orbital auto-escalation should stay disabled until the validation procedure in
  `docs/rf-real-acquisition.md` has been executed against real passes. The code now makes that
  procedure executable; it does not substitute for it.
- `PowerDbm` is only meaningful once `SDR_CALIBRATION_OFFSET_DB` has been measured against a known
  source. Until then `Calibrated` is false and the honest field is `PowerDbFS`.

## Suggested next step

Wire `dsp.CompareWithPrediction` into the existing `GET /api/v1/rf/doppler` endpoint so the
expected-versus-observed comparison the backend already computes from TLEs is fed by measured
carrier offsets rather than by the anomaly frequency field. That closes the loop between the
orbital model and the radio, which is the thing the platform claims to do.
