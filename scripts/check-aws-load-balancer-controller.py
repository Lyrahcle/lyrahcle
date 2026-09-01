#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
infra = root / "deploy" / "terraform" / "aws"
controller = infra / "aws-load-balancer-controller.tf"
versions = infra / "versions.tf"
variables = infra / "variables.tf"
guardrails = infra / "guardrails.tf"
outputs = infra / "outputs.tf"
tfvars = infra / "terraform.tfvars.example"
aws_values = root / "deploy" / "helm" / "lyrahcle" / "values-aws.example.yaml"
required = [controller, versions, variables, guardrails, outputs, tfvars, aws_values]
missing = [str(path.relative_to(root)) for path in required if not path.is_file()]
if missing:
    raise SystemExit("AWS Load Balancer Controller contract: missing files: " + ", ".join(missing))

text = controller.read_text()
version_text = versions.read_text()
variable_text = variables.read_text()
guardrail_text = guardrails.read_text()
output_text = outputs.read_text()
example = tfvars.read_text()
values = aws_values.read_text()
checks = {
    "Helm provider pin": 'source  = "hashicorp/helm"' in version_text and 'version = "3.2.0"' in version_text,
    "Kubernetes provider pin": 'source  = "hashicorp/kubernetes"' in version_text and 'version = "3.2.1"' in version_text,
    "controller chart": 'chart            = "aws-load-balancer-controller"' in text,
    "official chart repository": 'repository       = "https://aws.github.io/eks-charts"' in text,
    "pinned chart variable": 'aws_load_balancer_controller_chart_version' in variable_text and 'default     = "1.14.0"' in variable_text,
    "Pod Identity trust": 'pods.eks.amazonaws.com' in text and 'sts:AssumeRole' in text and 'sts:TagSession' in text,
    "controller ServiceAccount": 'resource "kubernetes_service_account_v1" "aws_load_balancer_controller"' in text,
    "Pod Identity association": 'resource "aws_eks_pod_identity_association" "aws_load_balancer_controller"' in text,
    "Helm reuses ServiceAccount": 'name  = "serviceAccount.create"' in text and 'value = "false"' in text,
    "dedicated role": 'resource "aws_iam_role" "aws_load_balancer_controller"' in text,
    "dedicated policy": 'resource "aws_iam_policy" "aws_load_balancer_controller"' in text,
    "no node-role policy attachment": 'module.eks.eks_managed_node_groups' not in text,
    "EKS exec auth": '"eks",' in text and '"get-token",' in text and 'command     = "aws"' in text,
    "explicit region": 'name  = "region"' in text,
    "explicit VPC": 'name  = "vpcId"' in text,
    "HA replicas": 'aws_load_balancer_controller_replica_count' in variable_text and 'default     = 2' in variable_text,
    "production HA guardrail": 'Production AWS Load Balancer Controller deployments must run at least two replicas.' in guardrail_text,
    "controller outputs": 'output "aws_load_balancer_controller_role_arn"' in output_text,
    "example enabled": 'enable_aws_load_balancer_controller       = true' in example,
    "ALB class": 'className: alb' in values,
    "ALB IP targets": 'alb.ingress.kubernetes.io/target-type: ip' in values,
    "ACM certificate": 'alb.ingress.kubernetes.io/certificate-arn:' in values,
    "HTTPS redirect": "alb.ingress.kubernetes.io/ssl-redirect: '443'" in values,
    "NLB class": 'loadBalancerClass: service.k8s.aws/nlb' in values,
}
failed = [name for name, ok in checks.items() if not ok]
if failed:
    raise SystemExit("AWS Load Balancer Controller contract failed: " + ", ".join(failed))

if 'role       = module.eks' in text or 'role_arn = module.eks' in text:
    raise SystemExit("AWS Load Balancer Controller contract: controller permissions must not be attached to the EKS node role")

print("AWS Load Balancer Controller contract: OK")
