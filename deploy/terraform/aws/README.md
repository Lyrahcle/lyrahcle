# Lyrahcle AWS infrastructure (Terraform/OpenTofu)

This root module provisions the AWS foundation for a future Lyrahcle EKS
migration while keeping the application Helm chart cloud-neutral.

It creates:

- a multi-AZ VPC with public, private, database and ElastiCache subnets;
- an EKS cluster with AL2023 managed node groups and the Pod Identity agent;
- immutable ECR repositories for backend, frontend and CVE sync;
- encrypted private RDS PostgreSQL with AWS-managed master-password rotation;
- an encrypted Secrets Manager container for Lyrahcle application secrets;
- optional encrypted ElastiCache/Valkey, disabled until application Redis TLS is ready.

It deliberately does **not** create DNS, ACM certificates, the AWS Load Balancer
Controller, Prometheus, or the Lyrahcle Helm release. Those are cluster/add-on
and application deployment layers, not VPC/data-plane infrastructure.

## Tooling

Terraform >= 1.5.7 or a compatible OpenTofu release is expected. The AWS
provider and public AWS modules are downloaded during `init`.

```bash
cd deploy/terraform/aws
cp terraform.tfvars.example terraform.tfvars
terraform init -backend=false
terraform fmt -check -recursive
terraform validate
terraform plan -out=/tmp/lyrahcle-aws.tfplan
```

For production, configure remote encrypted state first:

```bash
cp backend.hcl.example backend.hcl
# edit backend.hcl
terraform init -backend-config=backend.hcl
```

Never commit `terraform.tfvars`, `backend.hcl`, plan files, state files, or real
secrets. Do commit the generated `.terraform.lock.hcl` after the first trusted
`init` so provider selections remain reproducible.

## Production guardrails

Production automatically forces:

- one NAT gateway per AZ;
- RDS Multi-AZ;
- RDS deletion protection;
- a final RDS snapshot;
- 14 days of database backups;
- an ElastiCache AUTH token if ElastiCache is enabled.

The EKS API public endpoint is allowed by default for staging convenience. Set
`cluster_endpoint_public_access_cidrs` to your VPN/office CIDR before applying a
production environment.

## ElastiCache is intentionally opt-in

The AWS resource is configured with KMS at-rest encryption and TLS in transit.
Lyrahcle's current Redis wiring remains address-based, so `create_redis=false`
is the safe default. Do not disable ElastiCache TLS to work around that gap.
Implement and test application Redis TLS/auth first, then enable this resource.
