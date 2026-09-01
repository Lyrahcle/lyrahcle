variable "aws_region" {
  description = "AWS region for the Lyrahcle environment."
  type        = string
  default     = "eu-west-1"
}

variable "name" {
  description = "Base resource name."
  type        = string
  default     = "lyrahcle"
}

variable "environment" {
  description = "Environment name such as staging or production."
  type        = string
  default     = "staging"

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production."
  }
}

variable "kubernetes_version" {
  description = "EKS Kubernetes minor version. Keep this within standard support."
  type        = string
  default     = "1.35"
}

variable "vpc_cidr" {
  description = "CIDR for the EKS VPC."
  type        = string
  default     = "10.42.0.0/16"
}

variable "az_count" {
  description = "Number of AZs. Production should use at least 3."
  type        = number
  default     = 3

  validation {
    condition     = var.az_count >= 2 && var.az_count <= 4
    error_message = "az_count must be between 2 and 4."
  }
}

variable "single_nat_gateway" {
  description = "Use one NAT gateway to reduce staging cost. Set false for production AZ resilience."
  type        = bool
  default     = true
}

variable "cluster_endpoint_public_access" {
  description = "Allow access to the EKS API public endpoint."
  type        = bool
  default     = true
}

variable "cluster_endpoint_public_access_cidrs" {
  description = "CIDRs allowed to reach the EKS public API endpoint. Never leave 0.0.0.0/0 in production."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "node_instance_types" {
  description = "Instance types for the primary EKS managed node group."
  type        = list(string)
  default     = ["m6i.large", "m6a.large", "m5.large"]
}

variable "node_min_size" {
  type    = number
  default = 2
}

variable "node_desired_size" {
  type    = number
  default = 2
}

variable "node_max_size" {
  type    = number
  default = 6
}

variable "postgres_instance_class" {
  description = "RDS PostgreSQL instance class."
  type        = string
  default     = "db.t4g.medium"
}

variable "postgres_engine_version" {
  description = "Optional PostgreSQL engine version. Null lets RDS select the current default for the major engine."
  type        = string
  default     = null
}

variable "postgres_allocated_storage_gib" {
  type    = number
  default = 40
}

variable "postgres_max_allocated_storage_gib" {
  type    = number
  default = 200
}

variable "postgres_multi_az" {
  description = "RDS Multi-AZ. Production is forced to true regardless of this value."
  type        = bool
  default     = false
}

variable "postgres_deletion_protection" {
  description = "Protect RDS from accidental deletion. Production is forced to true."
  type        = bool
  default     = false
}

variable "create_redis" {
  description = "Create encrypted ElastiCache/Valkey for Lyrahcle distributed state."
  type        = bool
  default     = false
}

variable "redis_engine" {
  description = "ElastiCache protocol-compatible engine."
  type        = string
  default     = "valkey"

  validation {
    condition     = contains(["valkey", "redis"], var.redis_engine)
    error_message = "redis_engine must be valkey or redis."
  }
}

variable "redis_node_type" {
  type    = string
  default = "cache.t4g.small"
}

variable "redis_auth_token" {
  description = "Optional ElastiCache AUTH token. Sensitive values supplied to Terraform are still stored in state."
  type        = string
  sensitive   = true
  default     = null

  validation {
    condition = var.redis_auth_token == null || (
      length(var.redis_auth_token) >= 16 &&
      length(var.redis_auth_token) <= 128 &&
      can(regex("^[A-Za-z0-9!&#$^<>-]+$", var.redis_auth_token))
    )
    error_message = "redis_auth_token must be 16-128 characters using AWS ElastiCache AUTH-compatible characters."
  }
}

variable "tags" {
  description = "Additional tags applied to AWS resources."
  type        = map(string)
  default     = {}
}
