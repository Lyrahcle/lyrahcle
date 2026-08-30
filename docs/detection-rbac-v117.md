# v11.7 Detection and operational RBAC completion

RF detection rules are tenant-local policy. Migration 0034 assigns legacy rules to
the default organization, seeds the built-in rules for every other organization,
and seeds them for organizations created in the future.

The API now uses:
- `detections.read` for detection-rule reads
- `detections.manage` for detection-rule mutations
- `alerts.read` for asset-alert reads
- `alerts.manage` for alert acknowledgement
- `orbital.read` for satellite list, stream, passes and orbital-event reads
- `orbital.manage` for satellite/orbital mutations

Correlation loads enabled rules only from the anomaly's organization and detection
matches carry organization ownership for defense in depth.
