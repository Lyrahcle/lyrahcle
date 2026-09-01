# AWS infrastructure as code

Lyrahcle v14.6.8 adds the first AWS infrastructure layer under
`deploy/terraform/aws`. It is intentionally separate from the cloud-neutral Helm
chart.

## Baseline architecture

```text
Internet / agents
       |
  ALB / NLB (Kubernetes-managed later)
       |
   Amazon EKS
   /    |    \
backend frontend cve-sync
   |        |
   +-- RDS PostgreSQL
   +-- ElastiCache/Valkey (after Redis TLS app support)

Images -> ECR
Secrets -> Secrets Manager + KMS
```

The EKS worker nodes run in private subnets. RDS and ElastiCache have separate
private subnet groups and security groups that accept traffic only from the EKS
node security group.

## Why managed node groups

Managed node groups keep EC2 lifecycle, draining and node upgrades in EKS while
remaining explicit and portable. The baseline uses AL2023 and a minimum of two
nodes. Pod Identity Agent is installed as an EKS add-on so future controllers or
workloads can receive IAM without static AWS credentials.

## Load balancers

The application chart already maps browser ingress to ALB and the optional
agent mTLS endpoint to NLB. The Terraform layer intentionally does not create
those LBs directly: the Kubernetes AWS Load Balancer Controller should reconcile
them from Ingress/Service objects.

Install the controller only after the EKS cluster exists, using its own
ServiceAccount and Pod Identity/IRSA role. Keep agent traffic in TLS passthrough
mode so the Go backend validates client certificates.

## Data services

RDS is private, encrypted with the environment KMS key, uses a managed master
password in Secrets Manager, and is forced Multi-AZ in production.

ElastiCache is present as an opt-in resource with at-rest and in-transit
encryption. It remains disabled by default until Lyrahcle has first-class Redis
TLS/auth support. The migration policy is to fix the client, not weaken the
managed service.

## Secrets

Terraform creates only the application Secrets Manager secret metadata. Secret
values must be populated outside Terraform so JWT/admin/SMTP tokens do not enter
Terraform configuration or plan output. A later deployment block can integrate
External Secrets/CSI using EKS Pod Identity while preserving the Helm
`existingSecret` contract.

## State

Terraform state is sensitive infrastructure data. Local state is acceptable only
for disposable experimentation. Production must use encrypted remote state with
versioning and locking, with access restricted to the deployment role. Commit
the generated `.terraform.lock.hcl`; do not commit state, plans or real tfvars.
