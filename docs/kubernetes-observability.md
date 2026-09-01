# Kubernetes staging and observability

Lyrahcle v14.6.7 extends the cloud-neutral Helm deployment with a reproducible
pre-production path and optional Prometheus Operator integration. The
application chart still does not install PostgreSQL, Redis, Prometheus, Grafana
or cloud-specific controllers.

## Staging overlay

`deploy/helm/lyrahcle/values-staging.yaml` is the portable pre-production
baseline. It runs two backend and two frontend replicas with PDBs and topology
spread enabled, while keeping HPA, ingress and internal TLS disabled so the same
values file can run in a disposable kind cluster.

This overlay is **not** a production-security profile. Production deployments
must continue to start from `values-production.yaml`.

Render it with:

```bash
helm lint deploy/helm/lyrahcle -f deploy/helm/lyrahcle/values-staging.yaml
helm template lyrahcle deploy/helm/lyrahcle \
  -f deploy/helm/lyrahcle/values-staging.yaml >/tmp/lyrahcle-staging.yaml
```

## Disposable local Kubernetes environment

The repository includes a three-node kind topology plus disposable PostgreSQL
and Redis Deployments under `deploy/k8s/local`. The data services use
`emptyDir`; deleting the kind cluster deletes all data. They exist only to
exercise the Kubernetes application lifecycle before an external/managed data
layer is available.

Prerequisites:

- Docker
- kind
- kubectl
- Helm
- Python 3

Create/build/deploy and run the smoke gate:

```bash
./scripts/k8s-local-up.sh
```

The script:

1. creates the kind cluster if needed;
2. creates the namespace and local Secrets;
3. starts disposable PostgreSQL and Redis;
4. builds backend, frontend and CVE-sync images;
5. loads the images into kind;
6. installs the Helm chart with `values-staging.yaml`;
7. waits for two backend and two frontend replicas;
8. runs `scripts/k8s-smoke-test.sh`.

Keep the cluster and rerun only the smoke test with:

```bash
./scripts/k8s-smoke-test.sh
```

Delete the whole disposable environment with:

```bash
./scripts/k8s-local-down.sh
```

Set `LYRAHCLE_LOCAL_ADMIN_PASSWORD` before the first local install to override
the deliberately obvious local-only bootstrap password. The script generates a
random JWT secret when it creates the application Secret and preserves the
Secret on subsequent runs.

## Metrics Services

The backend Service exposes its existing `/metrics` endpoint on the application
port. v14.6.7 also creates a dedicated ClusterIP Service for the CVE worker's
port `9091`.

For small development clusters, set:

```yaml
monitoring:
  prometheusAnnotations:
    enabled: true
```

This adds standard `prometheus.io/*` annotations. Annotation discovery is kept
off in the production overlay because HTTPS trust configuration is usually
better represented explicitly by the monitoring stack.

## Prometheus Operator

When the cluster has the `monitoring.coreos.com/v1` CRDs installed, enable:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    labels:
      release: kube-prometheus-stack
  prometheusRule:
    enabled: true
    labels:
      release: kube-prometheus-stack
```

The chart creates separate ServiceMonitors for backend and CVE sync. With
backend TLS enabled, scraping fails closed unless the monitor is given a CA
Secret or the operator explicitly chooses `insecureSkipVerify`. Production
values point at `lyrahcle-backend-ca` and keep `insecureSkipVerify: false`.

The monitor uses named Service ports, not numeric target-port strings, so it
follows the normal Prometheus Operator service-discovery contract.

## Alert rules

The optional PrometheusRule covers application signals that are useful in a
Kubernetes deployment:

- elevated backend 5xx rate;
- elevated backend p95 latency;
- fallback away from the distributed rate-limit backend;
- missing/overdue remote-config signing keys;
- durable background-job failures;
- distributed scheduler lock failures;
- CVE-sync errors;
- stale CVE synchronization.

Threshold values that reasonably vary by environment are Helm values. Routing,
paging destinations and Alertmanager configuration remain responsibilities of
the cluster monitoring layer.

## NetworkPolicy interaction

When `networkPolicy.enabled=true`, the backend policy already permits the
frontend. To scrape metrics from another namespace, also configure:

```yaml
networkPolicy:
  monitoring:
    enabled: true
    namespaceLabels:
      kubernetes.io/metadata.name: monitoring
    podLabels: {}
```

If the monitoring namespace uses a different label, change the selector. A
non-empty `podLabels` map narrows access further to matching Prometheus pods.

NetworkPolicy behavior still depends on the cluster CNI. Test the policy in the
actual staging cluster before enabling it in production.

## Future AWS/EKS mapping

Nothing in this block requires AWS. On EKS the same metrics Services can be
scraped by a Prometheus Operator stack, an OpenTelemetry/Prometheus collector,
or another managed collection path. Keep the application chart responsible for
exposing metrics and rules; keep IAM, remote-write credentials, workspace
creation and collector deployment in the future `infra/aws` layer.
