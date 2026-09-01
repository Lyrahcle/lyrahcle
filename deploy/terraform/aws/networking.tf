module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "6.7.0"

  name = local.cluster_name
  cidr = var.vpc_cidr
  azs  = local.azs

  private_subnets     = local.private_subnets
  public_subnets      = local.public_subnets
  database_subnets    = local.database_subnets
  elasticache_subnets = local.elasticache_subnets

  create_database_subnet_group    = true
  create_elasticache_subnet_group = true

  enable_nat_gateway = true
  single_nat_gateway = var.environment == "production" ? false : var.single_nat_gateway

  enable_dns_support   = true
  enable_dns_hostnames = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = "1"
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = "1"
  }

  tags = local.tags
}
