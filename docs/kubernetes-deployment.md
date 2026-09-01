# Kubernetes deployment

Lyrahcle ships a cloud-neutral Helm chart at `deploy/helm/lyrahcle`. The chart
packages the stateless control-plane workloads only: backend, frontend and the
CVE sync worker. PostgreSQL, Redis, public certificate management and secret
material stay external so the same chart can run on a local cluster, bare metal,
managed Kubernetes or EKS.

## What the chart provides

- Backend and frontend Deployments with rolling updates.
- Startup, readiness and liveness probes.
- Resource requests and limits.
- Optional HPA (`autoscaling/v2`) for backend and frontend.
- Optional PodDisruptionBudgets.
- Optional zone/hostname topology spread.
- Dedicated ServiceAccounts for backend, frontend and CVE sync with Kubernetes
  API token automount disabled.
- Optional Ingress for the browser UI.
- Optional direct `LoadBalancer` Service for SDR-agent mTLS traffic.
- Default-deny capable NetworkPolicy with explicit ingress and egress paths for frontend, backend, CVE sync, DNS, datastores, monitoring and external HTTPS.
- File-based application secret custody. Kubernetes Secret projections are
  copied into an in-memory `emptyDir` with mode `0400` because Lyrahcle's file
  secret provider deliberately rejects symlinks and loose permissions.
- Backend-to-frontend TLS verification support.
- Production validation that rejects plaintext backend service, mutable
  `latest` image tags and an agent LoadBalancer without mTLS.

## Multi-replica safety

Every backend replica starts by checking database migrations. `db.Migrate`
uses a PostgreSQL session advisory lock held on one acquired connection, so
only one replica applies migrations at a time. Other replicas wait, re-read
`schema_migrations`, and continue once the active migrator releases the lock.
The local test gate includes a concurrent migration regression test.

Recurring backend jobs already use the Redis distributed job lock. This means
backend Deployments can scale horizontally without intentionally duplicating
singleton scheduled jobs.

## Prerequisites

You need:

- Kubernetes 1.30 or newer.
- Helm 3.
- Reachable PostgreSQL and Redis services.
- Published backend, frontend and CVE-sync images.
- An existing Kubernetes Secret containing application secret values.

The chart does **not** install PostgreSQL or Redis. For production use managed or
HA data services. For development, a separate disposable database/Redis stack is
recommended instead of making the application chart own persistent data.

## Required application Secret

Create `lyrahcle-app-secrets` before the workloads start. Minimum keys are:

- `DATABASE_URL`
- `REDIS_URL`
- `JWT_SECRET`
- `SEED_ADMIN_PASSWORD`

Optional keys include `JWT_SECRET_PREVIOUS`, `SMTP_PASSWORD` and
`AGENT_CONFIG_SIGNER_TOKEN`.

For a disposable development namespace you can create the Secret directly:

```bash
kubectl -n lyrahcle create secret generic lyrahcle-app-secrets \
  --from-literal=DATABASE_URL='postgres://USER:PASSWORD@postgres.example:5432/lyrahcle?sslmode=require' \
  --from-literal=REDIS_URL='rediss://:<AUTH_TOKEN>@redis.example:6379/0' \
  --from-literal=JWT_SECRET='replace-with-at-least-32-random-bytes' \
  --from-literal=SEED_ADMIN_PASSWORD='replace-with-a-strong-password'
```

Do not use command-line literals for long-lived production secrets. Synchronize
an external secret manager into the Kubernetes Secret or mount an equivalent
provider under controlled operations.

## Development render

The default values are intentionally non-production and keep Ingress disabled:

```bash
helm lint deploy/helm/lyrahcle
helm template lyrahcle deploy/helm/lyrahcle > /tmp/lyrahcle.yaml
```

Install into a namespace after providing images and the Secret:

```bash
helm upgrade --install lyrahcle deploy/helm/lyrahcle \
  --namespace lyrahcle \
  --create-namespace \
  --set backend.image.repository=registry.example/lyrahcle-backend \
  --set backend.image.tag=14.6.7 \
  --set frontend.image.repository=registry.example/lyrahcle-frontend \
  --set frontend.image.tag=14.6.7 \
  --set cveSync.image.repository=registry.example/lyrahcle-cve-sync \
  --set cveSync.image.tag=14.6.7
```

Without Ingress, access the UI with:

```bash
kubectl -n lyrahcle port-forward service/lyrahcle-frontend 5173:80
```


## Staging and local Kubernetes smoke

Use `values-staging.yaml` to exercise the multi-replica application lifecycle
without production-only TLS/Ingress dependencies. A three-node kind topology,
disposable PostgreSQL/Redis manifests and build/deploy/smoke scripts are included
for this purpose:

```bash
./scripts/k8s-local-up.sh
./scripts/k8s-smoke-test.sh
./scripts/k8s-local-down.sh
```

The local data layer is intentionally ephemeral and must never be treated as a
production PostgreSQL/Redis design. See `docs/kubernetes-observability.md` for
the complete staging workflow and Prometheus integration.

## Production baseline

Start from `values-production.yaml`:

```bash
helm lint deploy/helm/lyrahcle \
  -f deploy/helm/lyrahcle/values-production.yaml

helm upgrade --install lyrahcle deploy/helm/lyrahcle \
  --namespace lyrahcle \
  --create-namespace \
  -f deploy/helm/lyrahcle/values-production.yaml \
  -f values-site.yaml
```

The production overlay enables two replicas, HPA, PDB, topology spread, internal backend TLS and default-deny NetworkPolicy. Replace the example hostname, image repositories, ingress-controller selectors and datastore CIDRs in a site-specific values file.

### Backend TLS

Lyrahcle itself refuses plaintext backend HTTP when `ENV=production`. Supply a
TLS Secret named by `backend.tls.existingSecret` with `tls.crt` and `tls.key`.
The frontend should verify that certificate using a CA Secret containing
`ca.crt`.

The public Ingress TLS certificate and the internal backend certificate are
separate concerns. A cloud load balancer may terminate public TLS while the
frontend still talks HTTPS to the backend inside the cluster.

## SDR agent endpoint

Normal browser/API traffic enters through the frontend. SDR agents that require
end-to-end client-certificate validation can use the optional `agentService`.
It points directly at the backend and must only be enabled together with
`backend.agentMtls.enabled=true`.

Keep this endpoint at TCP/TLS passthrough when the backend must inspect and
validate the client certificate itself.

## Availability controls

For a production cluster:

- Run at least two backend and frontend replicas.
- Enable topology spreading across zones and nodes.
- Enable PDBs after confirming the cluster has enough capacity.
- Install Metrics Server (or a compatible metrics API) before enabling HPA.
- Set resource requests based on measured load, not the chart defaults.
- Keep PostgreSQL and Redis outside the application Deployment lifecycle.
- Verify database/Redis connection pool sizing as replicas scale.


## Kubernetes metrics and alerting

The chart exposes the backend metrics Service and a CVE-sync metrics Service.
Small clusters can use optional Prometheus scrape annotations. Clusters running
Prometheus Operator can instead enable the chart's `ServiceMonitor` and
`PrometheusRule` resources. Backend HTTPS scraping supports CA-secret
verification and fails Helm validation when TLS is enabled without either a CA
or an explicit insecure override.

See `docs/kubernetes-observability.md` for values, NetworkPolicy interaction and
the local smoke workflow.

## Pod Security Admission

The application workloads are compatible with the Kubernetes `restricted` Pod Security Standard: they run as non-root, use the runtime-default seccomp profile, drop all Linux capabilities, disable privilege escalation, use read-only root filesystems and do not share host network, PID or IPC namespaces. The backend secret-preparation init container follows the same policy.

The local kind workflow labels the namespace with `warn` and `audit` at the restricted level because the disposable PostgreSQL and Redis development images are outside the application chart. Production namespaces should enforce the policy after their managed datastores are external:

```bash
./scripts/k8s-enforce-restricted.sh
```

`deploy/k8s/security/namespace-restricted.yaml` contains the equivalent namespace labels for the standard `lyrahcle` namespace.

## NetworkPolicy

`networkPolicy.enabled` stays false in base values so development clusters remain portable, but production and AWS overlays enable it. When enabled, Lyrahcle applies default-deny ingress and egress to its own Pods and adds only the required paths: DNS, frontend to backend, backend/CVE sync to PostgreSQL and Redis, outbound HTTPS, monitoring, configured ingress sources and optional direct SDR-agent mTLS sources.

For AWS, `values-aws.example.yaml` scopes ALB-to-frontend and RDS/ElastiCache egress to the example `10.42.0.0/16` VPC. Change those CIDRs if `vpc_cidr` changes. Enabling the direct agent LoadBalancer while NetworkPolicy is active requires explicit `networkPolicy.agentIngressCidrs`; Helm rejects an empty allowlist. Extra egress rules remain explicit through the component-specific `*ExtraEgress` values.

## Local validation

The normal project gate validates the Kubernetes contracts:

```bash
./scripts/test-local-v1.sh
```

If `helm` is installed, the script also runs Helm parser/lint checks. GitHub CI
always installs the pinned Helm version and renders both production and AWS
example overlays.
