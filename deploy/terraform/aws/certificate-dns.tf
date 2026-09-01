resource "aws_acm_certificate" "public" {
  count = var.create_public_certificate ? 1 : 0

  domain_name       = var.public_domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = local.tags
}

resource "aws_route53_record" "public_certificate_validation" {
  for_each = var.create_public_certificate ? {
    for option in aws_acm_certificate.public[0].domain_validation_options : option.domain_name => {
      name   = option.resource_record_name
      record = option.resource_record_value
      type   = option.resource_record_type
    }
  } : {}

  allow_overwrite = true
  zone_id         = var.route53_zone_id
  name            = each.value.name
  type            = each.value.type
  ttl             = 60
  records         = [each.value.record]
}

resource "aws_acm_certificate_validation" "public" {
  count = var.create_public_certificate ? 1 : 0

  certificate_arn         = aws_acm_certificate.public[0].arn
  validation_record_fqdns = [for record in aws_route53_record.public_certificate_validation : record.fqdn]
}
