# Core telemetry visualizations

The Core UI now visualizes only telemetry already returned by existing APIs:

- RF power trend
- RF spectrum snapshot
- RF station coverage from configured SDR station coordinates
- anomaly intensity (`|z-score|`) over time
- incident severity distribution
- incident creation timeline
- CVE severity distribution
- CVE publication timeline
- live orbital altitude profile

No chart invents or interpolates measurements that are not present in the API.

Doppler and pass-elevation curves are intentionally deferred until the frontend
receives time-series pass prediction samples from a dedicated API. They should not
be approximated in the browser from unrelated live-position data.
