# Satellite pass prediction (v9.3)

Lyrahcle predicts ground-station visibility windows from the same stored TLE and propagation path used by orbital-aware RF correlation. The API reports acquisition of signal (AOS), loss of signal (LOS), peak time, maximum elevation, and pass duration.

## Endpoint

`GET /api/v1/satellites/{satelliteID}/passes`

Requires role `viewer` or higher.

Required query parameters:

- `latitude_deg`: ground-station latitude in `[-90, 90]`.
- `longitude_deg`: ground-station longitude in `[-180, 180]`.

Optional query parameters:

- `altitude_m`: station altitude, default `0`, accepted range `[-500, 10000]`.
- `from`: RFC3339 UTC-compatible start time, default current time.
- `hours`: prediction horizon, default `24`, maximum `72`.
- `min_elevation_deg`: visibility mask, default `0`, range `[0, 90]`.
- `limit`: maximum returned windows, default `10`, maximum `50`.

Example:

```text
GET /api/v1/satellites/<uuid>/passes?latitude_deg=40.4168&longitude_deg=-3.7038&altitude_m=667&hours=24&min_elevation_deg=10
```

Each pass includes:

- `aos` and `los`;
- `peak_at`;
- `max_elevation_deg`;
- `duration_seconds`;
- `aos_truncated` when the satellite was already above the requested mask at the start of the window;
- `los_truncated` when it remains above the mask at the end of the window.

## TLE safety

Pass prediction uses the v9.2.2 TLE-quality policy. `fresh` TLEs return high orbital confidence and `aging` TLEs return degraded confidence. `stale` and `unusable` TLEs are rejected with HTTP `422` rather than producing operationally misleading AOS/LOS times.

This does not change the orbital incident-severity gate: `ORBITAL_AUTO_ESCALATION_ENABLED` remains disabled by default until representative physical SDR validation is complete.

## Algorithm

The predictor scans the requested horizon in 30-second coarse intervals, detects threshold crossings, refines AOS/LOS to approximately one second, and independently refines the elevation peak within each pass. It reuses the existing `orbit.PositionAt` and `orbitalcontext.Evaluate` path so pass scheduling and RF/orbital correlation share propagation and station geometry.

## Operational notes

Predicted visibility is geometric. Local obstruction masks, antenna patterns, RF interference, receiver sensitivity, atmospheric effects, and hardware tuning latency are outside the current model. Use `min_elevation_deg` to apply a practical station mask and treat aging-TLE results as degraded planning evidence.
