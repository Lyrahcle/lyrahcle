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
  }
}
