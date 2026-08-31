# Agent hardware capabilities

Lyrahcle agents report SDR hardware inventory through the machine-authenticated `POST /api/v1/agents/me/device` endpoint. The backend derives the tenant from the authenticated agent credential; callers cannot select another organization in the payload.

The v13.10 report extends the original driver/model/serial fields with architecture, OS, driver/firmware versions, frequency range, sample rates, bandwidths, gain elements, channel count, GPS/PPS, temperature and sample-drop counters. Reports are stored as JSONB beside the existing device metadata so Fleet can consume the same contract later without a second inventory model.

The UI labels devices as `compatible`, `degraded`, `unsupported` or `stale`. `compatible` only means that the self-reported software capabilities meet Lyrahcle's v1 baseline. It does **not** mean the hardware has completed the physical RTL-SDR/HackRF lab-validation gate.

Reports older than 15 minutes are `stale`. Known SDR drivers with incomplete capability data are `degraded`; unknown or RX-incapable devices are `unsupported`.
