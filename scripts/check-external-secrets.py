#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
infra = root / "deploy" / "terraform" / "aws"
tf = infra / "external-secrets.tf"
variables = infra / "variables.tf"
outputs = infra / "outputs.tf"
secrets = infra / "secrets.tf"
values = root / "deploy" / "helm" / "lyrahcle" / "values.yaml"
aws_values = root / "deploy" / "helm" / "lyrahcle" / "values-aws.example.yaml"
template = root / "deploy" / "helm" / "lyrahcle" / "templates" / "external-secret.yaml"
validate = root / "deploy" / "helm" / "lyrahcle" / "templates" / "validate.yaml"
readme = root / "README.md"
screenshot = root / "docs" / "screenshots" / "dashboard.png"
required = [tf, variables, outputs, secrets, values, aws_values, template, validate, readme, screenshot]
missing = [str(path.relative_to(root)) for path in required if not path.is_file()]
if missing:
    raise SystemExit("External Secrets contract: missing files: " + ", ".join(missing))

text = tf.read_text()
variable_text = variables.read_text()
output_text = outputs.read_text()
secret_text = secrets.read_text()
value_text = values.read_text()
aws_value_text = aws_values.read_text()
template_text = template.read_text()
validate_text = validate.read_text()
readme_text = readme.read_text()
checks = {
    "official chart repository": 'repository = "https://charts.external-secrets.io"' in text,
    "pinned chart": 'default     = "2.10.0"' in variable_text,
    "Pod Identity trust": 'pods.eks.amazonaws.com' in text and 'sts:AssumeRole' in text and 'sts:TagSession' in text,
    "dedicated ServiceAccount": 'resource "kubernetes_service_account_v1" "external_secrets"' in text,
    "Pod Identity association": 'resource "aws_eks_pod_identity_association" "external_secrets"' in text,
    "read-only secret actions": 'secretsmanager:GetSecretValue' in text and 'secretsmanager:DescribeSecret' in text,
    "no secret mutation actions": all(token not in text for token in ["secretsmanager:PutSecretValue", "secretsmanager:UpdateSecret", "secretsmanager:DeleteSecret"]),
    "scoped secret ARN": 'resources = [aws_secretsmanager_secret.app.arn]' in text,
    "KMS decrypt": 'actions   = ["kms:Decrypt"]' in text and 'resources = [aws_kms_key.data.arn]' in text,
    "Terraform stores no secret value": 'aws_secretsmanager_secret_version' not in secret_text and 'aws_secretsmanager_secret_version' not in text,
    "base disabled": 'externalSecrets:\n  enabled: false' in value_text,
    "AWS enabled": 'externalSecrets:\n  enabled: true' in aws_value_text,
    "SecretStore v1": 'apiVersion: external-secrets.io/v1\nkind: SecretStore' in template_text,
    "ExternalSecret v1": 'apiVersion: external-secrets.io/v1\nkind: ExternalSecret' in template_text,
    "Secrets Manager provider": 'service: SecretsManager' in template_text,
    "no static AWS credentials": 'accessKeyIDSecretRef' not in template_text and 'secretAccessKeySecretRef' not in template_text,
    "target contract": 'targetSecretName: lyrahcle-app-secrets' in value_text and 'creationPolicy: Owner' in template_text,
    "extract structured secret": 'dataFrom:' in template_text and 'extract:' in template_text,
    "fail closed region": 'externalSecrets.region is required when External Secrets is enabled' in validate_text,
    "fail closed key": 'externalSecrets.remoteSecretKey is required when External Secrets is enabled' in validate_text,
    "operator output": 'output "external_secrets_role_arn"' in output_text,
    "README screenshot": './docs/screenshots/dashboard.png' in readme_text and screenshot.stat().st_size > 10000,
}
failed = [name for name, ok in checks.items() if not ok]
if failed:
    raise SystemExit("External Secrets contract failed: " + ", ".join(failed))
print("External Secrets contract: OK")
