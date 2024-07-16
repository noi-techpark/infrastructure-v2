resource "aws_route53_zone" "main" {
    name = var.main-zone
    force_destroy = true
}

resource "aws_route53_record" "eks-ingress" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "eks-ingress"
  type    = "A"
  ttl     = "300"
  records = [aws_eip.eks-ingress-b.public_ip]
}

locals {
  prod = var.ENVIRONMENT == "prod"
  service_domains_dev = [
    "mobility.api",
    "tourism.api",
    "push.api",
    "analytics",
    "rabbitmq",
    "spreadsheets"
  ]
  service_domains_prod = [
    "push.api",
  ]
  service_domains = local.prod ? local.service_domains_prod : local.service_domains_dev
}

resource "aws_route53_record" "eks-service-domains" {
  for_each = toset(local.service_domains)
  zone_id = aws_route53_zone.main.zone_id
  name    = each.value
  type    = "CNAME"
  ttl     = "300"
  records = [aws_route53_record.eks-ingress.fqdn]
}
