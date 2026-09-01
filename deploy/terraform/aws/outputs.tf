output "cluster_name" {
  value = module.eks.cluster_name
}

output "cluster_endpoint" {
  value     = module.eks.cluster_endpoint
  sensitive = true
}

output "vpc_id" {
  value = module.vpc.vpc_id
}

output "private_subnet_ids" {
  value = module.vpc.private_subnets
}

output "ecr_repository_urls" {
  value = { for name, repo in aws_ecr_repository.app : name => repo.repository_url }
}

output "postgres_endpoint" {
  value = aws_db_instance.postgres.address
}

output "postgres_master_secret_arn" {
  value     = try(aws_db_instance.postgres.master_user_secret[0].secret_arn, null)
  sensitive = true
}

output "redis_primary_endpoint" {
  value = var.create_redis ? aws_elasticache_replication_group.redis[0].primary_endpoint_address : null
}

output "redis_tls_endpoint_url" {
  value = var.create_redis ? "rediss://${aws_elasticache_replication_group.redis[0].primary_endpoint_address}:6379/0" : null
}

output "application_secret_arn" {
  value = aws_secretsmanager_secret.app.arn
}

output "kubectl_config_command" {
  value = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}

output "aws_load_balancer_controller_role_arn" {
  value = var.enable_aws_load_balancer_controller ? aws_iam_role.aws_load_balancer_controller[0].arn : null
}

output "aws_load_balancer_controller_service_account" {
  value = var.enable_aws_load_balancer_controller ? var.aws_load_balancer_controller_service_account : null
}

output "aws_load_balancer_controller_chart_version" {
  value = var.enable_aws_load_balancer_controller ? var.aws_load_balancer_controller_chart_version : null
}

output "external_secrets_role_arn" {
  value = try(aws_iam_role.external_secrets[0].arn, null)
}

output "external_secrets_service_account" {
  value = var.enable_external_secrets_operator ? "${var.external_secrets_namespace}/${var.external_secrets_service_account}" : null
}


output "aws_region" {
  value = var.aws_region
}

output "vpc_cidr" {
  value = var.vpc_cidr
}

output "application_secret_name" {
  value = aws_secretsmanager_secret.app.name
}

output "public_domain_name" {
  value = var.public_domain_name
}

output "public_certificate_arn" {
  value = var.create_public_certificate ? aws_acm_certificate_validation.public[0].certificate_arn : null
}

output "external_dns_role_arn" {
  value = var.enable_external_dns ? aws_iam_role.external_dns[0].arn : null
}

output "external_dns_service_account" {
  value = var.enable_external_dns ? "${var.external_dns_namespace}/${var.external_dns_service_account}" : null
}
