#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INFRA="$ROOT/deploy/terraform/aws"

for command_name in aws kubectl helm; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "ERROR: missing required command: $command_name" >&2
    exit 1
  fi
done

if command -v tofu >/dev/null 2>&1; then
  IAC=tofu
elif command -v terraform >/dev/null 2>&1; then
  IAC=terraform
else
  echo "ERROR: install OpenTofu or Terraform first." >&2
  exit 1
fi

cd "$INFRA"
aws sts get-caller-identity >/dev/null
REGION="$($IAC output -raw aws_region)"
CLUSTER="$($IAC output -raw cluster_name)"
CERTIFICATE="$($IAC output -raw public_certificate_arn)"
SECRET_ARN="$($IAC output -raw application_secret_arn)"
aws eks update-kubeconfig --region "$REGION" --name "$CLUSTER" >/dev/null

aws acm describe-certificate --region "$REGION" --certificate-arn "$CERTIFICATE" --query 'Certificate.Status' --output text | grep -qx ISSUED
aws secretsmanager describe-secret --region "$REGION" --secret-id "$SECRET_ARN" >/dev/null
kubectl -n kube-system rollout status deployment/aws-load-balancer-controller --timeout=120s
kubectl -n external-secrets rollout status deployment/external-secrets --timeout=120s
if kubectl -n external-dns get deployment external-dns >/dev/null 2>&1; then
  kubectl -n external-dns rollout status deployment/external-dns --timeout=120s
else
  echo "ERROR: external-dns deployment is not installed." >&2
  exit 1
fi

echo "AWS production readiness: OK"
