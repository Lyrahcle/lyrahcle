# AWS production readiness

This block closes the remaining platform wiring between the AWS Terraform root and the cloud-neutral Lyrahcle Helm chart without applying infrastructure automatically.

## Public endpoint

The AWS root can create an ACM certificate for an explicitly supplied public DNS name and validate it through an existing Route53 public hosted zone. It never creates a hosted zone implicitly.

Set `public_domain_name`, `route53_zone_id`, `create_public_certificate=true` and `enable_external_dns=true`. Production guardrails require the managed certificate and DNS publishing path.

The public request path is:

```text
Browser
  |
Route53
  |
ALB :443 + ACM
  |
Frontend
  |
Backend HTTPS
```

The AWS Load Balancer Controller reconciles the ALB from the Lyrahcle `Ingress`. ExternalDNS observes the same Ingress and publishes its hostname in the configured Route53 zone. Both controllers use dedicated EKS Pod Identity roles.

## ExternalDNS boundary

ExternalDNS is optional outside production. Its IAM role can change records only in the configured hosted zone. Discovery/list operations remain read-only. The chart is pinned, uses the AWS provider, `upsert-only` policy, TXT ownership and both domain and hosted-zone filters.

The ExternalDNS namespace enforces the Kubernetes `restricted` Pod Security Standard. The chart receives explicit non-root, no-escalation, read-only-root-filesystem, dropped-capabilities and RuntimeDefault seccomp settings.

## Image release flow

ECR repositories use immutable tags. `scripts/aws-build-push-images.sh` refuses to reuse an existing tag, builds backend/frontend/CVE sync images, pushes them and resolves the digest returned by ECR.

A normal release candidate can use the current commit as its immutable tag:

```bash
./scripts/aws-build-push-images.sh "$(git rev-parse --short=12 HEAD)"
```

The default output is `/tmp/lyrahcle-aws-images.json`. It contains only repository URLs, the release tag and image digests. It contains no application secrets.

After Terraform has created the EKS foundation, ACM certificate and controllers, render the dynamic Helm overlay:

```bash
./scripts/aws-render-release-values.py /tmp/lyrahcle-aws-images.json
```

The resulting `/tmp/lyrahcle-aws-release-values.yaml` reads non-secret Terraform outputs and pins each workload by digest. It also supplies the domain, ACM certificate ARN, Secrets Manager key, AWS region and VPC CIDR to the AWS Helm overlay.

Review before deployment:

```bash
helm template lyrahcle deploy/helm/lyrahcle \
  -f deploy/helm/lyrahcle/values-aws.example.yaml \
  -f /tmp/lyrahcle-aws-release-values.yaml >/tmp/lyrahcle-rendered.yaml
```

Do not commit `/tmp/lyrahcle-aws-images.json`, generated release values, Terraform state, backend configuration or secret values.

## Readiness check

Once an AWS staging environment exists, the read-only readiness helper verifies the active AWS identity, ACM issuance, application secret metadata and rollout status for the AWS Load Balancer Controller, External Secrets Operator and ExternalDNS:

```bash
./scripts/aws-production-readiness.sh
```

The helper does not read secret values and does not run `terraform apply` or `helm upgrade`.

## Deployment boundary

Infrastructure creation and application release remain separate deliberate actions:

1. `terraform plan` and review the AWS infrastructure change.
2. `terraform apply` only when the infrastructure change is approved.
3. Populate the Secrets Manager JSON value out-of-band.
4. Build and push immutable application images.
5. Render and review the Helm values overlay.
6. Enforce the `restricted` namespace policy before the production application release.
7. Run `helm upgrade --install` only after the rendered resources and image digests are approved.
8. Verify ALB health, DNS, TLS, workloads and security E2E checks before cutover.

No Terraform resource in this block deploys the Lyrahcle application release itself. This keeps infrastructure changes, secret population and application rollout independently reviewable.
