data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  cluster_name = "${var.name}-${var.environment}"
  azs          = slice(data.aws_availability_zones.available.names, 0, var.az_count)

  private_subnets     = [for i in range(var.az_count) : cidrsubnet(var.vpc_cidr, 4, i)]
  public_subnets      = [for i in range(var.az_count) : cidrsubnet(var.vpc_cidr, 4, i + 4)]
  database_subnets    = [for i in range(var.az_count) : cidrsubnet(var.vpc_cidr, 5, i + 16)]
  elasticache_subnets = [for i in range(var.az_count) : cidrsubnet(var.vpc_cidr, 5, i + 20)]

  tags = merge({
    Project     = "Lyrahcle"
    Environment = var.environment
    ManagedBy   = "Terraform/OpenTofu"
  }, var.tags)
}
