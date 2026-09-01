#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
infra = root / "deploy" / "terraform" / "aws"
certificate = infra / "certificate-dns.tf"
external_dns = infra / "external-dns.tf"
variables = infra / "variables.tf"
guardrails = infra / "guardrails.tf"
outputs = infra / "outputs.tf"
ecr = infra / "ecr.tf"
values = root / "deploy" / "helm" / "lyrahcle" / "values-aws.example.yaml"
build = root / "scripts" / "aws-build-push-images.sh"
render = root / "scripts" / "aws-render-release-values.py"
readiness = root / "scripts" / "aws-production-readiness.sh"
docs = root / "docs" / "aws-production-readiness.md"
required = [certificate, external_dns, variables, guardrails, outputs, ecr, values, build, render, readiness, docs]
missing = [str(path.relative_to(root)) for path in required if not path.is_file()]
if missing:
    raise SystemExit("AWS production readiness contract: missing files: " + ", ".join(missing))

certificate_text = certificate.read_text()
dns_text = external_dns.read_text()
variable_text = variables.read_text()
guardrail_text = guardrails.read_text()
output_text = outputs.read_text()
ecr_text = ecr.read_text()
value_text = values.read_text()
build_text = build.read_text()
render_text = render.read_text()
readiness_text = readiness.read_text()
checks = {
    "ACM DNS certificate": 'resource "aws_acm_certificate" "public"' in certificate_text and 'validation_method = "DNS"' in certificate_text,
    "Route53 validation": 'resource "aws_route53_record" "public_certificate_validation"' in certificate_text and 'aws_acm_certificate_validation' in certificate_text,
    "existing hosted zone input": 'variable "route53_zone_id"' in variable_text and 'variable "public_domain_name"' in variable_text,
    "ExternalDNS chart pin": 'default     = "1.21.1"' in variable_text and 'repository = "https://kubernetes-sigs.github.io/external-dns/"' in dns_text,
    "ExternalDNS Pod Identity": 'pods.eks.amazonaws.com' in dns_text and 'aws_eks_pod_identity_association" "external_dns' in dns_text,
    "Route53 scoped mutation": 'route53:ChangeResourceRecordSets' in dns_text and 'hostedzone/${coalesce(var.route53_zone_id' in dns_text,
    "domain and zone filters": 'domainFilters = compact([var.public_domain_name])' in dns_text and '--zone-id-filter=' in dns_text,
    "TXT ownership": 'registry      = "txt"' in dns_text and 'txtOwnerId    = local.cluster_name' in dns_text,
    "restricted ExternalDNS": 'pod-security.kubernetes.io/enforce' in dns_text and 'readOnlyRootFilesystem' in dns_text and 'drop = ["ALL"]' in dns_text,
    "ACM input guardrail": 'ACM certificate creation requires public_domain_name and route53_zone_id.' in guardrail_text,
    "ExternalDNS input guardrail": 'ExternalDNS requires public_domain_name and route53_zone_id.' in guardrail_text,
    "Terraform block closure syntax": "}  }" not in guardrail_text,
    "public domain regex HCL-safe": 'regex("^[A-Za-z0-9.-]+[.][A-Za-z]{2,}$"' in variable_text,
    "ExternalDNS version regex HCL-safe": 'regex("^[0-9]+[.][0-9]+[.][0-9]+$"' in variable_text,
    "certificate output": 'output "public_certificate_arn"' in output_text,
    "region and release outputs": 'output "aws_region"' in output_text and 'output "application_secret_name"' in output_text,
    "ECR immutable": 'image_tag_mutability = "IMMUTABLE"' in ecr_text,
    "build refuses tag reuse": 'immutable ECR tag already exists' in build_text and 'aws ecr describe-images' in build_text,
    "digest capture": 'imageDetails[0].imageDigest' in build_text and 'sha256:' in build_text,
    "render digest pin": 'digest:' in render_text and 'public_certificate_arn' in render_text and 'application_secret_name' in render_text,
    "ALB HTTPS placeholder": 'alb.ingress.kubernetes.io/certificate-arn:' in value_text and 'alb.ingress.kubernetes.io/ssl-redirect:' in value_text,
    "ExternalDNS hostname annotation": 'external-dns.alpha.kubernetes.io/hostname:' in value_text,
    "readiness is read only": 'terraform apply' not in readiness_text and 'helm upgrade' not in readiness_text and 'get-secret-value' not in readiness_text,
}
failed = [name for name, ok in checks.items() if not ok]
if failed:
    raise SystemExit("AWS production readiness contract failed: " + ", ".join(failed))
print("AWS production readiness contract: OK")
