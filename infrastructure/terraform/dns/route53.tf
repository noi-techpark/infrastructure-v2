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
    "analytics.beta",
    "neogy.ocpi.io",
    "driwe.ocpi.io",
    "google.spreadsheets.io",
    "prometheus",
    "tempo",
    "raw",
  ]
  static_domains_dev = [
    "docs",
  ]
  service_domains_prod = [
    "push.api",
    "google.spreadsheets.io",
    "neogy.ocpi.io",
  ]
  static_domains_prod = [
    "docs",
  ]
  service_domains = local.prod ? local.service_domains_prod : local.service_domains_dev
  static_domains = local.prod ? local.static_domains_prod : local.static_domains_dev
  reverse_proxy_ip = local.prod ? "proxy02.opendatahub.bz.it" : "proxy.testingmachine.eu"
}

resource "aws_route53_record" "eks-service-domains" {
  for_each = toset(local.service_domains)
  zone_id = aws_route53_zone.main.zone_id
  name    = each.value
  type    = "CNAME"
  ttl     = "300"
  records = [aws_route53_record.eks-ingress.fqdn]
}

resource "aws_route53_record" "static-domains" {
  for_each = toset(local.static_domains)
  zone_id = aws_route53_zone.main.zone_id
  name    = each.value
  type    = "CNAME"
  ttl     = "300"
  records = [local.reverse_proxy_ip]
}
