# AWS EKS migration

The Kubernetes chart is intentionally cloud-neutral. AWS should be introduced
as an infrastructure overlay rather than through AWS-specific code inside the
Lyrahcle application.

Use `deploy/helm/lyrahcle/values-aws.example.yaml` as the starting mapping, not
as a ready-to-run production configuration.

## Target service mapping

| Lyrahcle dependency | Kubernetes abstraction | AWS target |
| --- | --- | --- |
| Backend/frontend/CVE sync | Deployment | EKS worker capacity |
| Container images | OCI image | Amazon ECR |
| PostgreSQL | External endpoint | Amazon RDS/Aurora PostgreSQL |
| Redis | External endpoint | Amazon ElastiCache for Redis/Valkey |
| Browser ingress | Ingress | AWS Load Balancer Controller + ALB |
| Agent mTLS ingress | LoadBalancer Service | NLB with IP targets/TLS passthrough |
| Public TLS | Ingress/LB configuration | ACM certificate |
| Application secrets | Existing Kubernetes Secret | Secrets Manager integration |
| Pod AWS permissions | ServiceAccount | EKS Pod Identity |
| Metrics/traces | OpenTelemetry/Prometheus | Managed or self-hosted observability |

## Why the chart uses separate ServiceAccounts

Backend, frontend and CVE sync use different ServiceAccounts and do not mount a
Kubernetes API token by default. If an AWS integration later needs IAM, associate
only that component's ServiceAccount with the required IAM role. EKS Pod
Identity can provide temporary AWS credentials without adding AWS credential
logic to the application.

The initial migration should keep IAM permissions minimal. The frontend should
normally need no AWS API permissions at all.

## Ingress model

### Browser UI

The AWS example uses an ALB Ingress with IP targets. Public TLS terminates at the
ALB using ACM. The ALB sends HTTP to the frontend Service; the frontend then
proxies `/api/` to the backend over separately verified internal HTTPS.

### SDR agents

If SDR agents need a directly reachable endpoint for Lyrahcle-managed mTLS,
enable `agentService`. The AWS example uses the standard
`service.k8s.aws/nlb` LoadBalancerClass plus NLB IP targets. Keep TLS as
passthrough so the Go backend remains the component that validates agent client
certificates.

Do not expose the direct agent Service until `backend.agentMtls.enabled` is
configured with the agent CA material. The Helm chart rejects that unsafe
combination.

## Data layer

Move data services independently from the application:

1. Provision PostgreSQL with multi-AZ/HA, backups and encryption.
2. Restore a Lyrahcle backup into the managed database.
3. Point `DATABASE_URL` at the managed endpoint.
4. Provision Redis/Valkey with the availability mode required by the workload.
5. Populate `REDIS_URL` with the authenticated `rediss://` URI for the managed endpoint.
6. Run the full local/CI integration suite against a staging copy before cutover.

The backend migration advisory lock allows multiple EKS replicas to start at the
same time while still serializing schema migration execution.

### Redis TLS/auth readiness

Lyrahcle now supports authenticated `rediss://` connections in both the backend and CVE worker. Production startup rejects plaintext Redis URLs and unauthenticated Redis URLs. The client verifies certificates, enforces TLS 1.2 or newer, supports an optional custom CA and redacts credentials from logs and errors. The Terraform ElastiCache replication group can therefore remain encrypted in transit with TLS required rather than weakening the managed service configuration.

### Phase 1 — Kubernetes portability

Use the base chart outside AWS. Validate rolling upgrades, multiple backend
replicas, migration serialization, probes and graceful termination.

### Phase 2 — AWS staging

Create EKS, ECR, RDS, Redis/Valkey, ALB/NLB integration, DNS, ACM and external
secret management. Deploy the same application chart using the AWS values
overlay.

### Phase 3 — resilience

Exercise node termination, Availability Zone loss assumptions, backend rolling
updates, Redis interruption, database failover, backup restore and agent
reconnect behaviour. Record observed RPO/RTO.

### Phase 4 — production cutover

Pin images by immutable digest, apply least-privilege IAM, enable production
alerts, perform a final restore rehearsal and cut traffic only after the staging
security/E2E gates are green.

## Infrastructure-as-code boundary

v14.6.8 adds the AWS infrastructure root under `deploy/terraform/aws`. It now
creates the VPC, EKS managed-node foundation, ECR, RDS, KMS, Secrets Manager and
an optional encrypted ElastiCache replication group. It still does **not**
provision ALB/NLB resources directly; those remain reconciled from the same
cloud-neutral Helm Ingress/Service objects by the AWS Load Balancer Controller.

The ElastiCache resource remains disabled by default until the application Redis
client exposes and verifies TLS/auth. Do not weaken ElastiCache in-transit
encryption to bypass that application gap.

See `docs/aws-infrastructure-as-code.md` for plan/apply boundaries and guardrails.
