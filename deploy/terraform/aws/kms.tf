resource "aws_kms_key" "data" {
  description             = "Lyrahcle ${var.environment} managed-data encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true
  multi_region            = false

  tags = local.tags
}

resource "aws_kms_alias" "data" {
  name          = "alias/${local.cluster_name}-data"
  target_key_id = aws_kms_key.data.key_id
}
