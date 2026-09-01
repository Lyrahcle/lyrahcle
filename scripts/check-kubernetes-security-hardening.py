#!/usr/bin/env python3
from pathlib import Path
import sys


def check(root: Path) -> list[str]:
    files = {
        'backend': root / 'deploy/helm/lyrahcle/templates/backend-deployment.yaml',
        'frontend': root / 'deploy/helm/lyrahcle/templates/frontend-deployment.yaml',
        'cve': root / 'deploy/helm/lyrahcle/templates/cve-sync-deployment.yaml',
        'network': root / 'deploy/helm/lyrahcle/templates/networkpolicy.yaml',
        'values': root / 'deploy/helm/lyrahcle/values.yaml',
        'production': root / 'deploy/helm/lyrahcle/values-production.yaml',
        'aws': root / 'deploy/helm/lyrahcle/values-aws.example.yaml',
        'validation': root / 'deploy/helm/lyrahcle/templates/validate.yaml',
        'namespace': root / 'deploy/k8s/security/namespace-restricted.yaml',
        'local_up': root / 'scripts/k8s-local-up.sh',
        'enforce': root / 'scripts/k8s-enforce-restricted.sh',
    }
    errors = []
    for name, path in files.items():
        if not path.is_file():
            errors.append(f'missing {name}: {path.relative_to(root)}')
    if errors:
        return errors
    text = {name: path.read_text() for name, path in files.items()}

    def require(name: str, needle: str, label: str, minimum: int = 1) -> None:
        count = text[name].count(needle)
        if count < minimum:
            errors.append(f'{label}: expected {minimum}, found {count}')

    for name in ('backend', 'frontend', 'cve'):
        require(name, 'enableServiceLinks: false', f'{name} service-link isolation')
        require(name, 'hostNetwork: false', f'{name} host network isolation')
        require(name, 'hostPID: false', f'{name} host PID isolation')
        require(name, 'hostIPC: false', f'{name} host IPC isolation')
        require(name, 'fsGroupChangePolicy: OnRootMismatch', f'{name} fsGroup policy')
        require(name, 'privileged: false', f'{name} privileged guard')
        require(name, 'allowPrivilegeEscalation: false', f'{name} privilege escalation guard')
        require(name, 'readOnlyRootFilesystem: true', f'{name} read-only root filesystem')
        require(name, 'drop: ["ALL"]', f'{name} capability drop')
        require(name, 'type: RuntimeDefault', f'{name} seccomp profile')
    require('backend', 'drop: ["ALL"]', 'backend init and app capability drop', 2)
    if 'chown ' in text['backend']:
        errors.append('backend restricted init container must not depend on chown capability')

    require('network', 'name: {{ include "lyrahcle.fullname" . }}-default-deny', 'default deny policy')
    require('network', 'policyTypes: ["Ingress", "Egress"]', 'bidirectional policy isolation', 4)
    require('network', 'ingress: []', 'explicit ingress deny')
    require('network', 'egress: []', 'explicit egress deny')
    require('network', 'app.kubernetes.io/component: frontend', 'frontend policy')
    require('network', 'app.kubernetes.io/component: backend', 'backend policy')
    require('network', 'app.kubernetes.io/component: cve-sync', 'CVE sync policy')
    require('network', 'protocol: UDP\n          port: 53', 'UDP DNS egress')
    require('network', 'protocol: TCP\n          port: 53', 'TCP DNS egress')
    require('network', 'app.kubernetes.io/name: postgres', 'PostgreSQL local egress')
    require('network', 'app.kubernetes.io/name: redis', 'Redis local egress')
    require('network', '.Values.networkPolicy.externalDatastoreCidrs', 'external datastore CIDR egress')
    require('network', '.Values.networkPolicy.httpsEgress.cidrs', 'HTTPS egress allowlist')
    require('network', '.Values.networkPolicy.agentIngressCidrs', 'agent ingress CIDR allowlist')

    require('values', 'defaultDeny: true', 'default deny default')
    require('values', 'externalDatastoreCidrs: []', 'explicit external datastore allowlist')
    require('values', 'backendExtraEgress: []', 'backend escape hatch')
    require('production', 'networkPolicy:\n  enabled: true', 'production network policy enforcement')
    require('aws', 'networkPolicy:\n  enabled: true', 'AWS network policy enforcement')
    require('aws', 'frontendIngressCidrs:\n    - 10.42.0.0/16', 'AWS ALB VPC ingress scope')
    require('aws', 'externalDatastoreCidrs:\n    - 10.42.0.0/16', 'AWS datastore VPC scope')

    for mode in ('enforce', 'audit', 'warn'):
        require('namespace', f'pod-security.kubernetes.io/{mode}: restricted', f'namespace PSS {mode}')
        require('namespace', f'pod-security.kubernetes.io/{mode}-version: latest', f'namespace PSS {mode} version')
    require('local_up', 'pod-security.kubernetes.io/audit=restricted', 'local PSS audit')
    require('local_up', 'pod-security.kubernetes.io/warn=restricted', 'local PSS warning')
    if 'pod-security.kubernetes.io/enforce=restricted' in text['local_up']:
        errors.append('local kind bootstrap must not enforce restricted on development datastore pods')
    require('enforce', 'pod-security.kubernetes.io/enforce=restricted', 'production PSS enforcement helper')
    require('validation', 'production requires networkPolicy.enabled=true', 'production network policy validation')
    require('validation', 'networkPolicy requires agent ingress CIDRs', 'agent ingress validation')
    return errors


def main() -> int:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
    errors = check(root)
    if errors:
        for error in errors:
            print(f'kubernetes security hardening contract: {error}', file=sys.stderr)
        return 1
    print('kubernetes security hardening contract: OK')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
