data "aws_iam_policy_document" "external_secrets_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole", "sts:TagSession"]

    principals {
      type        = "Service"
      identifiers = ["pods.eks.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "external_secrets" {
  statement {
    effect = "Allow"
    actions = [
      "secretsmanager:DescribeSecret",
      "secretsmanager:GetSecretValue",
    ]
    resources = [aws_secretsmanager_secret.app.arn]
  }

  statement {
    effect    = "Allow"
    actions   = ["kms:Decrypt"]
    resources = [aws_kms_key.data.arn]
  }
}

resource "aws_iam_policy" "external_secrets" {
  count = var.enable_external_secrets_operator ? 1 : 0

  name        = "${local.cluster_name}-external-secrets"
  description = "Read-only access to Lyrahcle application secrets for External Secrets Operator."
  policy      = data.aws_iam_policy_document.external_secrets.json
  tags        = local.tags
}

resource "aws_iam_role" "external_secrets" {
  count = var.enable_external_secrets_operator ? 1 : 0

  name               = "${local.cluster_name}-external-secrets"
  assume_role_policy = data.aws_iam_policy_document.external_secrets_assume_role.json
  tags               = local.tags
}

resource "aws_iam_role_policy_attachment" "external_secrets" {
  count = var.enable_external_secrets_operator ? 1 : 0

  role       = aws_iam_role.external_secrets[0].name
  policy_arn = aws_iam_policy.external_secrets[0].arn
}

resource "kubernetes_namespace_v1" "external_secrets" {
  count = var.enable_external_secrets_operator ? 1 : 0

  metadata {
    name = var.external_secrets_namespace
    labels = {
      "pod-security.kubernetes.io/enforce"         = "restricted"
      "pod-security.kubernetes.io/enforce-version" = "latest"
      "pod-security.kubernetes.io/audit"           = "restricted"
      "pod-security.kubernetes.io/warn"            = "restricted"
    }
  }

  depends_on = [module.eks]
}

resource "kubernetes_service_account_v1" "external_secrets" {
  count = var.enable_external_secrets_operator ? 1 : 0

  metadata {
    name      = var.external_secrets_service_account
    namespace = kubernetes_namespace_v1.external_secrets[0].metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "external-secrets"
    }
  }
}

resource "aws_eks_pod_identity_association" "external_secrets" {
  count = var.enable_external_secrets_operator ? 1 : 0

  cluster_name    = module.eks.cluster_name
  namespace       = kubernetes_namespace_v1.external_secrets[0].metadata[0].name
  service_account = kubernetes_service_account_v1.external_secrets[0].metadata[0].name
  role_arn        = aws_iam_role.external_secrets[0].arn

  depends_on = [aws_iam_role_policy_attachment.external_secrets]
}

resource "helm_release" "external_secrets" {
  count = var.enable_external_secrets_operator ? 1 : 0

  name       = "external-secrets"
  namespace  = kubernetes_namespace_v1.external_secrets[0].metadata[0].name
  repository = "https://charts.external-secrets.io"
  chart      = "external-secrets"
  version    = var.external_secrets_chart_version

  atomic          = true
  cleanup_on_fail = true
  wait            = true
  timeout         = 600

  set = [
    {
      name  = "installCRDs"
      value = "true"
    },
    {
      name  = "serviceAccount.create"
      value = "false"
    },
    {
      name  = "serviceAccount.name"
      value = kubernetes_service_account_v1.external_secrets[0].metadata[0].name
    },
  ]

  depends_on = [aws_eks_pod_identity_association.external_secrets]
}
