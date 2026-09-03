# Real RF acquisition and spectral analysis

Until now the SDR agent shipped a single `soapy.MockDevice` whose `ReadSamples` returned a
deterministic ramp, and the only measurement was the mean power of that ramp. Every downstream
capability — waterfall, z-score anomalies, Doppler correlation, orbital escalation — was therefore
computed from synthetic data. This document describes the acquisition and analysis layers that
replace it, and the procedure that turns a real capture into the physical-validation evidence
required before orbital auto-escalation is enabled.

## Architecture

```
RTL-SDR ──USB──> rtl_tcp (Raspberry Pi) ──TCP──> RTLTCPDevice ──> SpectralSource ──> uplink
                                                        │
                                                        └──> forensics (.cf32) ──> replay.SpectralWindows
```

`internal/soapy` gained a second `Device` implementation, `RTLTCPDevice`, that speaks the `rtl_tcp`
wire protocol directly. There is no cgo and no external dependency, so the agent still
cross-compiles to `linux/arm` for the Raspberry Pi with `CGO_ENABLED=0` and still ships in a
`scratch` image.

`internal/dsp` is new and holds the analysis primitives: a radix-2 FFT, the four standard window
functions with their coherent gain and equivalent noise bandwidth, a Welch periodogram, and carrier
frequency estimation.

`internal/capture.SpectralSource` binds the two together and produces a `Reading` carrying real
spectral metrics instead of a single scalar.

## Selecting a device

The agent chooses its device from `SDR_DEVICE`:

| Value | Behaviour |
| --- | --- |
| unset or `mock://` | Mock device. The agent logs a warning stating that the output is synthetic and must not be used as detection evidence. |
| `rtltcp://host:port` | Real RTL-SDR through an `rtl_tcp` server. |

Optional query parameters on the `rtltcp` URI: `direct_sampling=0|1|2` for HF reception through the
I or Q branch, and `offset_tuning=1` to move the DC spike away from the centre of the band.

### Environment variables

| Variable | Default | Meaning |
| --- | --- | --- |
| `SDR_DEVICE` | `mock://` | Device URI. |
| `SDR_SAMPLE_RATE_HZ` | `2048000` | RTL2832U accepts 225001–3200000; 2.4 MS/s and above drop samples on a Pi 3B+. |
| `SDR_GAIN_DB` | `30` | Manual tuner gain. Ignored when `SDR_AUTO_GAIN` is set. |
| `SDR_AUTO_GAIN` | unset | Enables the tuner AGC. Never use it for measurements you intend to compare over time. |
| `SDR_FREQ_CORRECTION_PPM` | `0` | Crystal correction. See calibration below. |
| `SDR_BIAS_TEE` | unset | Powers an external LNA. |
| `SDR_FFT_SIZE` | `4096` | Must be a power of two. |
| `SDR_FFT_WINDOW` | `hann` | `rectangular`, `hann`, `hamming` or `blackman-harris`. |
| `SDR_FFT_OVERLAP` | `0.5` | Welch segment overlap. |
| `SDR_SAMPLES_PER_READING` | `32768` | Samples analysed per reading. Must be at least `SDR_FFT_SIZE`. |
| `SDR_CALIBRATION_OFFSET_DB` | `0` | dBFS to dBm offset. See calibration below. |
| `SDR_RECONNECT_ATTEMPTS` | `3` | Reconnects to `rtl_tcp` after a read failure. |

## Running rtl_tcp on the Raspberry Pi

```
sudo apt install rtl-sdr
rtl_tcp -a 0.0.0.0 -p 1234 -d 0
```

`rtl_tcp` has no authentication and no transport security, so it must never be exposed beyond the
loopback interface or a link the agent already protects. The supported deployment is `rtl_tcp`
bound to `127.0.0.1` on the same Pi as the agent, with the agent's mTLS uplink carrying anything
that leaves the device:

```
rtl_tcp -a 127.0.0.1 -p 1234 -d 0
SDR_DEVICE=rtltcp://127.0.0.1:1234 ./lyrahcle-sdr-agent
```

If the SDR is on a different host from the agent, tunnel the TCP session (WireGuard or an SSH local
forward) rather than binding `rtl_tcp` to a routable address.

## What a reading now contains

`capture.Reading.Spectral` carries:

- `PowerDbFS` — integrated power across the analysed band, referenced to full scale.
- `NoiseFloorDbFS` — median bin power, a robust floor estimate that a narrowband carrier cannot skew.
- `PeakPowerDbFS` and `SNRDb` — strongest bin and its margin over the floor.
- `CarrierOffsetHz` and `CarrierFrequencyHz` — estimated carrier position.
- `OccupiedBandwidthHz` — 99% power bandwidth.
- `ResolutionBandwidthHz` — equivalent noise bandwidth of one bin for the configured window, which
  is what a measurement has to be normalised by before it is comparable across FFT sizes.
- `EstimatorMethod` — `psd-peak`, `psd-peak-interpolated` or `psd-peak+kay`.
- `CalibrationOffsetDb` and `Calibrated`.

The `/api/v1/rf/ingest` payload is unchanged: the backend rejects unknown JSON fields, so extending
the wire format requires a coordinated backend change. Until then the spectral metrics are used
locally, logged, and available offline through the replay path.

## Frequency estimation

Estimation runs in two stages. The coarse stage takes the strongest PSD bin and refines it with
quadratic interpolation over its two neighbours, which is accurate to a fraction of a bin for a
windowed tone. When the SNR exceeds 6 dB the signal is mixed down by the coarse estimate and Kay's
weighted phase-difference estimator produces the residual, giving an offset far below the bin width.
If the residual exceeds one bin the fine stage is discarded and the coarse value stands, so a
mis-locked estimator degrades to the PSD peak rather than reporting a confident wrong answer.

`dsp.RangeRateFromOffsetMps` converts an offset to range rate using `v = -c·Δf/f₀`. A positive
Doppler offset means the emitter is approaching, so the range rate is negative. At 437.5 MHz a LEO
pass with a 7 km/s closing rate produces roughly +10.2 kHz.

## Calibration: dBFS is not dBm

An RTL-SDR reports no absolute power. Everything the DSP layer computes is referenced to converter
full scale, which is why `PowerDbFS` is the honest field and `PowerDbm` is only meaningful once
`SDR_CALIBRATION_OFFSET_DB` has been measured. `Calibrated` is false until then, and readings
should not be presented as absolute power while it is false.

To measure the offset, inject a signal of known level at the antenna port with a signal generator
or a calibrated noise source, record `PowerDbFS`, and set the offset to
`known_dBm − measured_dBFS`. The result is valid only for that gain setting, that frequency and
that RF chain; changing the gain, adding an LNA or changing the feedline invalidates it.

For ppm correction, tune to a stable reference — a GSM downlink or a local beacon of known
frequency — read `CarrierOffsetHz`, and set
`SDR_FREQ_CORRECTION_PPM = round(1e6 · offset_Hz / centre_Hz)`. Repeat after the dongle has reached
thermal equilibrium, since RTL-SDR crystals drift several ppm during the first minutes.

## Offline analysis of stored captures

Captures written by `IQ_CAPTURE_DIR` can be re-analysed without hardware:

```
REPLAY_FILE=agent-1_1756900000000000000.cf32 \
REPLAY_ROOT=/var/lib/lyrahcle/captures \
REPLAY_SPECTRAL=1 \
SDR_FFT_SIZE=4096 \
./lyrahcle-sdr-agent
```

`replay.SpectralWindows` produces one reading per analysis window with the full spectral metric set,
and `replay.DopplerTrackFromReadings` turns those into a range-rate track filtered by an SNR gate.
This is the evidence path: the same code that runs live also runs over an archived capture, so a
correlation result can be reproduced from the stored IQ file.

## Physical validation procedure

This is the gate that must pass before orbital auto-escalation is enabled.

1. Pick a satellite with a strong, well-documented beacon and a current TLE. NOAA APT at
   137.1/137.9125 MHz and the amateur 435–438 MHz band are the practical choices for an RTL-SDR.
2. Calibrate ppm and record the offset used.
3. Capture the whole pass to disk with `IQ_CAPTURE_DIR`, tuned to the nominal beacon frequency, gain
   fixed and AGC off. Record the ground station position and the TLE epoch alongside the capture.
4. Run the replay path with `REPLAY_SPECTRAL=1` and build the Doppler track with an SNR gate of at
   least 10 dB.
5. Propagate the same TLE for the same interval from the same station position and compute the
   predicted range rate, then the predicted Doppler with `dsp.OffsetFromRangeRateHz`.
6. Compare the two with `dsp.CompareWithPrediction`. Acceptance criteria for a pass above 20°
   elevation: residual within ±500 Hz at 437 MHz across at least 80% of the gated windows, no
   systematic sign error, and a residual curve that does not diverge monotonically. A constant
   offset across the whole pass indicates uncorrected ppm error rather than a correlation failure.
7. Archive the capture, the metadata, the TLE, the predicted and observed tracks, and the residual
   summary. That bundle is the evidence artifact for the validation gate.

Two passes of the same satellite on different days, plus one pass of a different satellite, is the
minimum before the gate should be considered met.

## Known limitations

- Only the RTL-SDR family is supported. HackRF, USRP and Airspy need either their own `Device`
  implementation or a real SoapySDR binding, which would reintroduce cgo.
- `rtl_tcp` reads are synchronous and unbuffered relative to the dongle, so sustained sample rates
  above roughly 2.048 MS/s on a Raspberry Pi 3B+ will drop samples. Dropped samples corrupt the
  phase continuity that Kay's estimator relies on.
- Kay's estimator assumes a single dominant carrier. Two comparable carriers in the analysis band
  produce an estimate somewhere between them; the coarse peak is the safer value in that case.
- The DC spike inherent to the RTL2832U sits at the centre bin. Tune slightly off the carrier, or
  enable `offset_tuning=1`, when the signal of interest is close to the tuned frequency.
