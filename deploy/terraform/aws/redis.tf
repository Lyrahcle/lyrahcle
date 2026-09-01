resource "aws_security_group" "redis" {
  count = var.create_redis ? 1 : 0

  name        = "${local.cluster_name}-redis"
  description = "ElastiCache access from Lyrahcle EKS nodes only"
  vpc_id      = module.vpc.vpc_id

  tags = local.tags
}

resource "aws_vpc_security_group_ingress_rule" "redis_from_eks" {
  count = var.create_redis ? 1 : 0

  security_group_id            = aws_security_group.redis[0].id
  referenced_security_group_id = module.eks.node_security_group_id
  from_port                    = 6379
  to_port                      = 6379
  ip_protocol                  = "tcp"
}

resource "aws_elasticache_replication_group" "redis" {
  count = var.create_redis ? 1 : 0

  replication_group_id = substr("${local.cluster_name}-redis", 0, 40)
  description          = "Lyrahcle ${var.environment} distributed state"

  engine             = var.redis_engine
  node_type          = var.redis_node_type
  port               = 6379
  num_cache_clusters = 2

  automatic_failover_enabled = true
  multi_az_enabled           = true

  subnet_group_name  = module.vpc.elasticache_subnet_group_name
  security_group_ids = [aws_security_group.redis[0].id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  kms_key_id                 = aws_kms_key.data.arn
  auth_token                 = var.redis_auth_token

  snapshot_retention_limit = var.environment == "production" ? 7 : 1
  apply_immediately        = var.environment != "production"

  lifecycle {
    precondition {
      condition     = var.environment != "production" || var.redis_auth_token != null
      error_message = "Production ElastiCache requires redis_auth_token. Keep create_redis=false until Lyrahcle Redis TLS/auth support is ready."
    }
  }

  tags = local.tags
}
