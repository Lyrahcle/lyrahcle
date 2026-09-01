#!/usr/bin/env python3
import argparse
import json
import shutil
import subprocess
from pathlib import Path


def output(iac, infra, name):
    return subprocess.check_output([iac, "output", "-raw", name], cwd=infra, text=True).strip()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("images")
    parser.add_argument("--output", default="/tmp/lyrahcle-aws-release-values.yaml")
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    infra = root / "deploy" / "terraform" / "aws"
    iac = shutil.which("tofu") or shutil.which("terraform")
    if not iac:
        raise SystemExit("install OpenTofu or Terraform first")
    images = json.loads(Path(args.images).read_text())
    domain = output(iac, infra, "public_domain_name")
    certificate = output(iac, infra, "public_certificate_arn")
    region = output(iac, infra, "aws_region")
    vpc_cidr = output(iac, infra, "vpc_cidr")
    secret_name = output(iac, infra, "application_secret_name")
    if not domain or domain == "null":
        raise SystemExit("public_domain_name Terraform output is required")
    if not certificate or certificate == "null":
        raise SystemExit("public_certificate_arn Terraform output is required")
    for key in ("backend", "frontend", "cve-sync"):
        digest = images.get(key, {}).get("digest", "")
        if not digest.startswith("sha256:"):
            raise SystemExit(f"missing immutable digest for {key}")
    text = f'''application:
  host: {domain}
  webauthnRpId: {domain}
  webauthnOrigins: https://{domain}
  passwordResetBaseUrl: https://{domain}/
backend:
  image:
    repository: {images["backend"]["repository"]}
    tag: {images["tag"]}
    digest: {images["backend"]["digest"]}
frontend:
  image:
    repository: {images["frontend"]["repository"]}
    tag: {images["tag"]}
    digest: {images["frontend"]["digest"]}
cveSync:
  image:
    repository: {images["cve-sync"]["repository"]}
    tag: {images["tag"]}
    digest: {images["cve-sync"]["digest"]}
ingress:
  annotations:
    alb.ingress.kubernetes.io/certificate-arn: '{certificate}'
    external-dns.alpha.kubernetes.io/hostname: '{domain}'
externalSecrets:
  region: {region}
  remoteSecretKey: {secret_name}
networkPolicy:
  frontendIngressCidrs:
    - {vpc_cidr}
  externalDatastoreCidrs:
    - {vpc_cidr}
'''
    Path(args.output).write_text(text)
    print(args.output)


if __name__ == "__main__":
    main()
