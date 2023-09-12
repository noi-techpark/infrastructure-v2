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
  service-domains = [
    "mobility.api",
    "tourism.api",
    "analytics",
    "rabbitmq"
  ]
}

resource "aws_route53_record" "eks-service-domains" {
  for_each = toset(local.service-domains)
  zone_id = aws_route53_zone.main.zone_id
  name    = each.value
  type    = "CNAME"
  ttl     = "300"
  records = [aws_route53_record.eks-ingress.fqdn]
}
