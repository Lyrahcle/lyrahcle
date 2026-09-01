resource "aws_security_group" "postgres" {
  name        = "${local.cluster_name}-postgres"
  description = "RDS PostgreSQL access from Lyrahcle EKS nodes only"
  vpc_id      = module.vpc.vpc_id

  tags = local.tags
}

resource "aws_vpc_security_group_ingress_rule" "postgres_from_eks" {
  security_group_id            = aws_security_group.postgres.id
  referenced_security_group_id = module.eks.node_security_group_id
  from_port                    = 5432
  to_port                      = 5432
  ip_protocol                  = "tcp"
}

resource "aws_db_instance" "postgres" {
  identifier = "${local.cluster_name}-postgres"

  engine         = "postgres"
  engine_version = var.postgres_engine_version
  instance_class = var.postgres_instance_class

  db_name  = "lyrahcle"
  username = "lyrahcle_admin"
  port     = 5432

  manage_master_user_password   = true
  master_user_secret_kms_key_id = aws_kms_key.data.arn

  allocated_storage     = var.postgres_allocated_storage_gib
  max_allocated_storage = var.postgres_max_allocated_storage_gib
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.data.arn

  db_subnet_group_name   = module.vpc.database_subnet_group_name
  vpc_security_group_ids = [aws_security_group.postgres.id]
  publicly_accessible    = false

  multi_az                     = var.environment == "production" ? true : var.postgres_multi_az
  backup_retention_period      = var.environment == "production" ? 14 : 7
  copy_tags_to_snapshot        = true
  auto_minor_version_upgrade   = true
  deletion_protection          = var.environment == "production" ? true : var.postgres_deletion_protection
  skip_final_snapshot          = var.environment != "production"
  final_snapshot_identifier    = var.environment == "production" ? "${local.cluster_name}-postgres-final" : null
  performance_insights_enabled = true

  tags = local.tags
}
