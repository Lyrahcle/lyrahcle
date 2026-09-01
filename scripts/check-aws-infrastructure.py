#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
infra = root / "deploy" / "terraform" / "aws"
required = [
    "versions.tf", "variables.tf", "locals.tf", "networking.tf", "eks.tf",
    "kms.tf", "ecr.tf", "rds.tf", "redis.tf", "secrets.tf", "outputs.tf",
    "guardrails.tf", "terraform.tfvars.example", "backend.hcl.example", "README.md",
]
missing = [name for name in required if not (infra / name).is_file()]
if missing:
    raise SystemExit(f"AWS IaC contract: missing files: {', '.join(missing)}")

text = "\n".join((infra / name).read_text() for name in required if name.endswith(".tf"))
checks = {
    "VPC module pin": 'terraform-aws-modules/vpc/aws' in text and 'version = "6.7.0"' in text,
    "EKS module pin": 'terraform-aws-modules/eks/aws' in text and 'version = "21.25.0"' in text,
    "Pod Identity agent": 'eks-pod-identity-agent' in text,
    "private RDS": 'publicly_accessible    = false' in text,
    "RDS encryption": 'storage_encrypted     = true' in text and 'manage_master_user_password' in text,
    "Redis TLS": 'transit_encryption_enabled = true' in text,
    "Redis at-rest encryption": 'at_rest_encryption_enabled = true' in text,
    "Redis opt-in": 'variable "create_redis"' in text and 'default     = false' in text,
    "ECR immutability": 'image_tag_mutability = "IMMUTABLE"' in text,
    "ECR scan": 'scan_on_push = true' in text,
    "Secrets Manager": 'aws_secretsmanager_secret' in text,
    "KMS rotation": 'enable_key_rotation     = true' in text,
    "production AZ guardrail": 'Production must span at least three availability zones.' in text,
    "production EKS CIDR guardrail": 'Production EKS API access must not allow 0.0.0.0/0.' in text,
    "node HA guardrail": 'EKS node sizing must keep at least two nodes' in text,
}
failed = [name for name, ok in checks.items() if not ok]
if failed:
    raise SystemExit("AWS IaC contract failed: " + ", ".join(failed))

example = (infra / "terraform.tfvars.example").read_text()
if 'create_redis = false' not in example:
    raise SystemExit("AWS IaC contract: example must keep encrypted Redis opt-in until client TLS ships")
if '0.0.0.0/0' in example:
    raise SystemExit("AWS IaC contract: tfvars example must not expose EKS API to the world")

backend = (infra / "backend.hcl.example").read_text()
if 'use_lockfile = true' not in backend:
    raise SystemExit("AWS IaC contract: S3 backend example must enable state locking")

print("AWS infrastructure-as-code contract: OK")
