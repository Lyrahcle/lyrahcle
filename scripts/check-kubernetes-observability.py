#!/usr/bin/env python3
"""Static contracts for v14.6.7 Kubernetes staging and monitoring integration."""
from __future__ import annotations

import os
import sys
from pathlib import Path


def check(root: Path) -> list[str]:
    errors: list[str] = []
    chart = root / "deploy/helm/lyrahcle"
    required = {
        "chart": chart / "Chart.yaml",
        "values": chart / "values.yaml",
        "staging": chart / "values-staging.yaml",
        "cve_service": chart / "templates/cve-sync-service.yaml",
        "service_monitor": chart / "templates/servicemonitor.yaml",
        "prom_rule": chart / "templates/prometheusrule.yaml",
        "network_policy": chart / "templates/networkpolicy.yaml",
        "validation": chart / "templates/validate.yaml",
        "kind": root / "deploy/kind/lyrahcle.yaml",
        "dependencies": root / "deploy/k8s/local/dependencies.yaml",
        "local_up": root / "scripts/k8s-local-up.sh",
        "local_down": root / "scripts/k8s-local-down.sh",
        "smoke": root / "scripts/k8s-smoke-test.sh",
    }
    for label, path in required.items():
        if not path.is_file():
            errors.append(f"missing {label}: {path}")
    if errors:
        return errors

    texts = {label: path.read_text() for label, path in required.items()}

    def require(label: str, needle: str, purpose: str) -> None:
        if needle not in texts[label]:
            errors.append(f"missing {purpose}: {needle!r}")

    require("chart", "appVersion:", "chart app version")
    require("chart", "version:", "chart package version")
    require("values", "serviceMonitor:", "ServiceMonitor values")
    require("values", "prometheusRule:", "PrometheusRule values")
    require("values", "prometheusAnnotations:", "annotation discovery values")
    require("staging", "replicaCount: 2", "multi-replica staging baseline")
    require("staging", "repository: lyrahcle/backend", "local backend image")
    require("staging", "repository: lyrahcle/frontend", "local frontend image")

    require("cve_service", "name: metrics", "named CVE metrics Service port")
    require("cve_service", 'prometheus.io/scrape: "true"', "optional CVE scrape annotations")
    require("service_monitor", "kind: ServiceMonitor", "Prometheus Operator monitor")
    require("service_monitor", "app.kubernetes.io/component: backend", "backend monitor selector")
    require("service_monitor", "app.kubernetes.io/component: cve-sync", "CVE monitor selector")
    require("service_monitor", "tlsConfig:", "backend TLS scrape configuration")
    require("service_monitor", "insecureSkipVerify:", "explicit TLS verification setting")

    for metric in (
        "lyrahcle_http_requests_total",
        "lyrahcle_http_request_duration_seconds_bucket",
        "lyrahcle_rate_limit_backend_fallback_total",
        "lyrahcle_signing_key_warnings",
        "lyrahcle_background_job_events_total",
        "lyrahcle_scheduled_job_leadership_total",
        "lyrahcle_cve_sync_errors_total",
        "lyrahcle_cve_sync_last_timestamp",
    ):
        require("prom_rule", metric, f"PrometheusRule metric {metric}")

    require("network_policy", ".Values.networkPolicy.monitoring.enabled", "monitoring NetworkPolicy gate")
    require("network_policy", "namespaceSelector:", "monitoring namespace selector")
    require("validation", "backend TLS ServiceMonitor requires a CA secret", "fail-closed ServiceMonitor TLS validation")

    require("kind", "role: control-plane", "kind control plane")
    if texts["kind"].count("role: worker") < 2:
        errors.append("kind staging cluster must define at least two workers")
    require("dependencies", "image: postgres:16-alpine", "local PostgreSQL dependency")
    require("dependencies", "image: redis:7-alpine", "local Redis dependency")
    require("dependencies", "emptyDir: {}", "disposable datastore storage")

    require("local_up", "kind load docker-image", "local image loading")
    require("local_up", "values-staging.yaml", "staging Helm overlay")
    require("local_up", "lyrahcle-app-secrets", "local application secret bootstrap")
    require("smoke", "/readyz", "backend readiness smoke")
    require("smoke", "/metrics", "metrics smoke")
    require("smoke", "-lt 2", "multi-replica smoke assertion")

    for script_key in ("local_up", "local_down", "smoke"):
        path = required[script_key]
        if os.name != "nt" and not os.access(path, os.X_OK):
            errors.append(f"script is not executable: {path}")

    return errors


def main() -> int:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors = check(root)
    if errors:
        for error in errors:
            print(f"kubernetes observability contract: {error}", file=sys.stderr)
        return 1
    print("kubernetes observability contract: OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
