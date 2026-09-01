resource "aws_secretsmanager_secret" "app" {
  name                    = "${local.cluster_name}/app"
  description             = "Lyrahcle application secret material. Populate out-of-band; Terraform creates metadata only."
  kms_key_id              = aws_kms_key.data.arn
  recovery_window_in_days = var.environment == "production" ? 30 : 7

  tags = local.tags
}
