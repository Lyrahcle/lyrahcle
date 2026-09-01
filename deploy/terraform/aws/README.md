# Lyrahcle AWS infrastructure (Terraform/OpenTofu)

This root module provisions the AWS foundation for a future Lyrahcle EKS
migration while keeping the application Helm chart cloud-neutral.

It creates:

- a multi-AZ VPC with public, private, database and ElastiCache subnets;
- an EKS cluster with AL2023 managed node groups and the Pod Identity agent;
- the AWS Load Balancer Controller with a dedicated IAM role, EKS Pod Identity association and pinned Helm release;
- immutable ECR repositories for backend, frontend and CVE sync;
- optional Route53 DNS validation for an ACM public certificate;
- optional ExternalDNS with a Route53-scoped EKS Pod Identity role;
- encrypted private RDS PostgreSQL with AWS-managed master-password rotation;
- an encrypted Secrets Manager container for Lyrahcle application secrets;
- External Secrets Operator with a dedicated read-only EKS Pod Identity role;
- optional encrypted ElastiCache/Valkey with TLS 1.2+ compatible client support and AUTH.

It deliberately does **not** create a Route53 hosted zone, Prometheus, application secret values, container images, or the Lyrahcle application Helm release. Public ACM/DNS automation is opt-in and always targets an explicitly supplied existing hosted zone. The AWS Load Balancer Controller, External Secrets Operator and ExternalDNS are EKS platform add-ons with dedicated Pod Identity roles.

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

## AWS Load Balancer Controller

`enable_aws_load_balancer_controller=true` installs the controller in `kube-system` through the official `eks-charts` Helm repository. The IAM role trusts only the EKS Pod Identity service and is associated only with the controller ServiceAccount. Node roles do not receive load-balancer controller permissions.

The chart version is pinned by `aws_load_balancer_controller_chart_version`. Upgrade it deliberately after reviewing the upstream release and IAM-policy changes. The Terraform runner needs the AWS CLI because the Helm provider authenticates to EKS with `aws eks get-token`. It also needs network reachability to the EKS API endpoint.

Set `public_domain_name`, `route53_zone_id`, `create_public_certificate=true` and `enable_external_dns=true` to automate ACM DNS validation and application hostname publication. Terraform still does not create the hosted zone itself. The repository-safe AWS Helm example is completed at release time by `scripts/aws-render-release-values.py`.

After apply, verify the controller with:

```bash
kubectl -n kube-system get deployment aws-load-balancer-controller
kubectl -n kube-system get pods -l app.kubernetes.io/name=aws-load-balancer-controller
```

## External Secrets Operator

`enable_external_secrets_operator=true` installs the pinned External Secrets Operator chart in the `external-secrets` namespace. Its ServiceAccount is associated with a dedicated EKS Pod Identity role that can read only the Terraform-created Lyrahcle application secret and decrypt it with the Lyrahcle data KMS key. Application workloads do not receive AWS secret-read permissions.

Terraform intentionally creates no `aws_secretsmanager_secret_version`. Populate the secret value out-of-band as a JSON object before deploying the Lyrahcle Helm chart with `externalSecrets.enabled=true`. The application chart then creates a namespaced `SecretStore` and `ExternalSecret` that materialize the existing `lyrahcle-app-secrets` contract. See `docs/aws-secrets-manager.md`.


## Production release bridge

`scripts/aws-build-push-images.sh` publishes unique immutable ECR tags and records the ECR digests. `scripts/aws-render-release-values.py` combines those digests with non-secret Terraform outputs to create a temporary Helm overlay containing the public domain, ACM certificate ARN, Secrets Manager key, AWS region and VPC CIDR. `scripts/aws-production-readiness.sh` performs read-only checks against an existing staging/production foundation and never reads secret values or applies changes. See `docs/aws-production-readiness.md`.
