# Lyrahcle AWS infrastructure (Terraform/OpenTofu)

This root module provisions the AWS foundation for a future Lyrahcle EKS
migration while keeping the application Helm chart cloud-neutral.

It creates:

- a multi-AZ VPC with public, private, database and ElastiCache subnets;
- an EKS cluster with AL2023 managed node groups and the Pod Identity agent;
- immutable ECR repositories for backend, frontend and CVE sync;
- encrypted private RDS PostgreSQL with AWS-managed master-password rotation;
- an encrypted Secrets Manager container for Lyrahcle application secrets;
- optional encrypted ElastiCache/Valkey with TLS 1.2+ compatible client support and AUTH.

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

The AWS resource is configured with KMS at-rest encryption, required TLS in transit and optional AUTH. Lyrahcle accepts `rediss://` URLs, verifies certificates with TLS 1.2 or newer and supports Redis usernames/passwords or ElastiCache AUTH tokens encoded in `REDIS_URL`. `create_redis=false` remains the default so infrastructure cost and secret population stay explicit. Production requires `redis_auth_token` when ElastiCache is enabled.

After apply, populate the application secret out-of-band with a URL such as `rediss://:<AUTH_TOKEN>@<PRIMARY_ENDPOINT>:6379/0`. The token must never be committed to Terraform examples, Helm values or source control. System trust roots are used by default; configure `REDIS_CA_CERT_FILE` only when a private CA is required.
