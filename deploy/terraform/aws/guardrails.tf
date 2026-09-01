resource "terraform_data" "guardrails" {
  lifecycle {
    precondition {
      condition     = var.environment != "production" || var.az_count >= 3
      error_message = "Production must span at least three availability zones."
    }

    precondition {
      condition     = var.environment != "production" || !contains(var.cluster_endpoint_public_access_cidrs, "0.0.0.0/0")
      error_message = "Production EKS API access must not allow 0.0.0.0/0. Restrict it to VPN/office CIDRs or disable public access."
    }

    precondition {
      condition     = var.node_min_size >= 2 && var.node_desired_size >= var.node_min_size && var.node_max_size >= var.node_desired_size
      error_message = "EKS node sizing must keep at least two nodes and satisfy min <= desired <= max."
    }

    precondition {
      condition     = var.environment != "production" || !var.enable_aws_load_balancer_controller || var.aws_load_balancer_controller_replica_count >= 2
      error_message = "Production AWS Load Balancer Controller deployments must run at least two replicas."
    }


    precondition {
      condition     = !var.create_public_certificate || (var.public_domain_name != null && var.route53_zone_id != null)
      error_message = "ACM certificate creation requires public_domain_name and route53_zone_id."
    }

    precondition {
      condition     = !var.enable_external_dns || (var.public_domain_name != null && var.route53_zone_id != null)
      error_message = "ExternalDNS requires public_domain_name and route53_zone_id."
    }
  }
}
