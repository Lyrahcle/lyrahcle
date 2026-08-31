#!/usr/bin/env python3
"""Static contract for the cloud-neutral Kubernetes deployment foundation."""
from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path


def check(root: Path) -> list[str]:
    errors: list[str] = []
    chart = root / "deploy/helm/lyrahcle"
    required_files = [
        "Chart.yaml", "values.yaml", "values-production.yaml", "values-aws.example.yaml",
        "templates/backend-deployment.yaml", "templates/frontend-deployment.yaml",
        "templates/cve-sync-deployment.yaml", "templates/backend-service.yaml",
        "templates/frontend-service.yaml", "templates/ingress.yaml", "templates/hpa.yaml",
        "templates/pdb.yaml", "templates/networkpolicy.yaml", "templates/serviceaccount.yaml",
        "templates/validate.yaml",
    ]
    for rel in required_files:
        if not (chart / rel).is_file():
            errors.append(f"missing {chart / rel}")
    if errors:
        return errors

    texts = {rel: (chart / rel).read_text() for rel in required_files}
    chart_yaml = texts["Chart.yaml"]
    values = texts["values.yaml"]
    production = texts["values-production.yaml"]
    aws = texts["values-aws.example.yaml"]
    backend = texts["templates/backend-deployment.yaml"]
    frontend = texts["templates/frontend-deployment.yaml"]
    cve = texts["templates/cve-sync-deployment.yaml"]
    services = texts["templates/backend-service.yaml"]
    hpa = texts["templates/hpa.yaml"]
    pdb = texts["templates/pdb.yaml"]
    service_accounts = texts["templates/serviceaccount.yaml"]
    validation = texts["templates/validate.yaml"]
    db_path = root / "apps/backend/internal/db/db.go"
    if not db_path.is_file():
        return [f"missing {db_path}"]
    db_go = db_path.read_text()

    def require(text: str, needle: str, label: str) -> None:
        if needle not in text:
            errors.append(f"missing {label}: {needle!r}")

    def reject(text: str, needle: str, label: str) -> None:
        if needle in text:
            errors.append(f"forbidden {label}: {needle!r}")

    require(chart_yaml, "apiVersion: v2", "Helm v3 chart API")
    require(chart_yaml, 'appVersion: "14.6.6"', "chart app version")
    require(values, "existingSecret: lyrahcle-app-secrets", "external application Secret")

    for name, text in (("backend", backend), ("frontend", frontend), ("cve-sync", cve)):
        require(text, "allowPrivilegeEscalation: false", f"{name} privilege-escalation guard")
        require(text, "seccompProfile:", f"{name} seccomp profile")
        require(text, "resources:", f"{name} resource controls")
        require(text, "automountServiceAccountToken: false", f"{name} API token opt-out")
    require(frontend, "runAsNonRoot: true", "frontend non-root runtime")
    require(frontend, "readOnlyRootFilesystem: true", "frontend read-only root filesystem")
    require(frontend, 'drop: ["ALL"]', "frontend capability drop")

    for probe in ("startupProbe:", "readinessProbe:", "livenessProbe:"):
        require(backend, probe, f"backend {probe[:-1]}")
        require(frontend, probe, f"frontend {probe[:-1]}")

    require(backend, "SECRET_PROVIDER", "file secret provider")
    require(backend, "prepare-secrets", "secret hardening init container")
    require(backend, "terminationGracePeriodSeconds: 30", "backend graceful shutdown budget")
    require(frontend, "BACKEND_TLS_VERIFY", "backend TLS verification switch")
    require(hpa, "apiVersion: autoscaling/v2", "HPA v2")
    require(pdb, "apiVersion: policy/v1", "PDB v1")
    require(service_accounts, "lyrahcle.backendServiceAccountName", "backend service account")
    require(service_accounts, "lyrahcle.frontendServiceAccountName", "frontend service account")
    require(service_accounts, "lyrahcle.cveSyncServiceAccountName", "CVE sync service account")

    require(production, "replicaCount: 2", "production replicas")
    reject(production, "tag: latest", "mutable production image tag")
    require(validation, "production must pin", "production image validation")
    require(validation, "agentService.enabled requires backend.agentMtls.enabled=true", "agent mTLS validation")

    require(aws, "className: alb", "AWS ALB ingress")
    require(aws, "alb.ingress.kubernetes.io/target-type: ip", "AWS ALB IP targets")
    require(aws, "loadBalancerClass: service.k8s.aws/nlb", "AWS NLB LoadBalancerClass")
    require(aws, "aws-load-balancer-nlb-target-type: ip", "AWS NLB IP targets")
    require(services, "loadBalancerClass:", "portable LoadBalancerClass field")

    require(db_go, "pg_advisory_lock", "serialized database migrations")
    require(db_go, "pg_advisory_unlock", "migration lock release")
    require(db_go, "pool.Acquire", "session-pinned migration lock")
    return errors


def main() -> int:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
    errors = check(root)
    if errors:
        for error in errors:
            print(f"kubernetes deployment contract: {error}", file=sys.stderr)
        return 1

    helm = shutil.which("helm")
    if helm:
        chart = root / "deploy/helm/lyrahcle"
        commands = [
            [helm, "lint", str(chart)],
            [helm, "lint", str(chart), "-f", str(chart / "values-production.yaml")],
            [helm, "template", "lyrahcle", str(chart), "-f", str(chart / "values-production.yaml")],
            [helm, "template", "lyrahcle", str(chart), "-f", str(chart / "values-aws.example.yaml")],
        ]
        for command in commands:
            subprocess.run(command, check=True, stdout=subprocess.DEVNULL)

    print("kubernetes deployment contract: OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
