data "aws_iam_policy_document" "external_dns_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole", "sts:TagSession"]

    principals {
      type        = "Service"
      identifiers = ["pods.eks.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "external_dns" {
  statement {
    effect = "Allow"
    actions = [
      "route53:ChangeResourceRecordSets",
      "route53:ListResourceRecordSets",
    ]
    resources = ["arn:${data.aws_partition.current.partition}:route53:::hostedzone/${coalesce(var.route53_zone_id, "disabled")}"]
  }

  statement {
    effect = "Allow"
    actions = [
      "route53:GetChange",
      "route53:ListHostedZones",
      "route53:ListTagsForResource",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  name        = "${local.cluster_name}-external-dns"
  description = "Route53 record management for Lyrahcle ExternalDNS."
  policy      = data.aws_iam_policy_document.external_dns.json
  tags        = local.tags
}

resource "aws_iam_role" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  name               = "${local.cluster_name}-external-dns"
  assume_role_policy = data.aws_iam_policy_document.external_dns_assume_role.json
  tags               = local.tags
}

resource "aws_iam_role_policy_attachment" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  role       = aws_iam_role.external_dns[0].name
  policy_arn = aws_iam_policy.external_dns[0].arn
}

resource "kubernetes_namespace_v1" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  metadata {
    name = var.external_dns_namespace
    labels = {
      "pod-security.kubernetes.io/enforce"         = "restricted"
      "pod-security.kubernetes.io/enforce-version" = "latest"
      "pod-security.kubernetes.io/audit"           = "restricted"
      "pod-security.kubernetes.io/warn"            = "restricted"
    }
  }

  depends_on = [module.eks]
}

resource "kubernetes_service_account_v1" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  metadata {
    name      = var.external_dns_service_account
    namespace = kubernetes_namespace_v1.external_dns[0].metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "external-dns"
    }
  }
}

resource "aws_eks_pod_identity_association" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  cluster_name    = module.eks.cluster_name
  namespace       = kubernetes_namespace_v1.external_dns[0].metadata[0].name
  service_account = kubernetes_service_account_v1.external_dns[0].metadata[0].name
  role_arn        = aws_iam_role.external_dns[0].arn

  depends_on = [aws_iam_role_policy_attachment.external_dns]
}

resource "helm_release" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  name       = "external-dns"
  namespace  = kubernetes_namespace_v1.external_dns[0].metadata[0].name
  repository = "https://kubernetes-sigs.github.io/external-dns/"
  chart      = "external-dns"
  version    = var.external_dns_chart_version

  atomic          = true
  cleanup_on_fail = true
  wait            = true
  timeout         = 600

  values = [yamlencode({
    provider = {
      name = "aws"
    }
    policy        = "upsert-only"
    registry      = "txt"
    txtOwnerId    = local.cluster_name
    sources       = ["ingress", "service"]
    domainFilters = compact([var.public_domain_name])
    extraArgs     = ["--zone-id-filter=${coalesce(var.route53_zone_id, "disabled")}"]
    podSecurityContext = {
      runAsNonRoot = true
      seccompProfile = {
        type = "RuntimeDefault"
      }
    }
    securityContext = {
      allowPrivilegeEscalation = false
      readOnlyRootFilesystem   = true
      capabilities = {
        drop = ["ALL"]
      }
    }
    serviceAccount = {
      create = false
      name   = kubernetes_service_account_v1.external_dns[0].metadata[0].name
    }
  })]

  depends_on = [aws_eks_pod_identity_association.external_dns]
}
