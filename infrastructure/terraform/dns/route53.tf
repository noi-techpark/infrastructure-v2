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

################################################################################
# -----> See below for new route management
################################################################################
locals {
  prod = var.ENVIRONMENT == "prod"
  service_domains_dev = [
    "mobility.api",
    "tourism.api",
    "push.api",
    "analytics",
    "rabbitmq",
    "neogy.ocpi.io",
    "driwe.ocpi.io",
    "sftp.io",
    "google.spreadsheets.io",
    "prometheus",
    "tempo",
    "files"
  ]
  static_domains_dev = [
    "docs",
  ]

  service_domains_prod = [
    "push.api",
    "google.spreadsheets.io",
    "neogy.ocpi.io",
    "driwe.ocpi.io",
    "sftp.io",
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

################################################################################
# New Route53 management
################################################################################
################################################################################
# Domain Matrix Per Environment
################################################################################

locals {
  is_prod = var.ENVIRONMENT == "prod"

  domain_matrix_by_zone = local.is_prod ? {
    # prod environment
    "obs.opendatahub.com" = [
      {
        domain  = "eks-ingress"
        target = "eip::public_ip"
        type   = "A"
      },
      {
        domain  = "prometheus"
        target = "eks-ingress.obs.opendatahub.com"
        type   = "CNAME"
      },
      {
        domain  = "tempo"
        target = "eks-ingress.obs.opendatahub.com"
        type   = "CNAME"
      }
    ],
    "internal.opendatahub.com" = [
      {
        domain  = "eks-ingress-private"
        target = "eip::private_ip"
        type   = "A"
      },
      {
        domain  = "authenticate"
        target = "eks-ingress-private.internal.opendatahub.com"
        type   = "CNAME"
      },
      {
        domain  = "raw"
        target = "eks-ingress-private.internal.opendatahub.com"
        type   = "CNAME"
      },
    ]
  } : {
    "internal.testingmachine.eu" = [
      {
        domain  = "eks-ingress-private"
        target = "eip::private_ip"
        type   = "A"
      },
      {
        domain  = "authenticate"
        target = "eks-ingress-private.internal.testingmachine.eu"
        type   = "CNAME"
      },
      {
        domain  = "raw"
        target = "eks-ingress-private.internal.testingmachine.eu"
        type   = "CNAME"
      }
    ]
  }

  # Flattened version (adds the `zone` to each rule)
  domain_matrix = flatten([
    for zone, records in local.domain_matrix_by_zone : [
      for r in records : merge(r, { zone = zone })
    ]
  ])
}

################################################################################
# Create Target Map for CNAME Lookups
################################################################################

locals {
  # A records that will be created as part of domain_matrix
  domain_a_fqdns = {
    for r in local.domain_matrix : "${r.domain}.${r.zone}" => "${r.domain}.${r.zone}" if r.type == "A"
  }

  # Final map of all targets used by CNAMEs
  target_map = merge(
    local.domain_a_fqdns,
    # Inject "special" targets
    {
      "ext::reverse-proxy" = local.is_prod ? "proxy02.opendatahub.bz.it" : "proxy.testingmachine.eu",
      "eip::public_ip" = aws_eip.eks-ingress-b.public_ip,
      "eip::private_ip" = aws_eip.eks-ingress-private.public_ip,
    }
  )
}

################################################################################
# Lookup Zones for CNAME Records
################################################################################

resource "aws_route53_zone" "cname_zones" {
  for_each = toset([
    for r in local.domain_matrix : r.zone
  ])

  name          = each.key
  force_destroy = true
}

################################################################################
# Expand CNAME Matrix with Resolved Targets and Zone IDs
################################################################################

locals {
  cname_matrix_expanded = [
    for record in local.domain_matrix : {
      key     = "${record.zone}_${record.domain}"
      zone_id = aws_route53_zone.cname_zones[record.zone].zone_id
      name    = record.domain
      type    = record.type
      target  = lookup(local.target_map, record.target, record.target)
    }
  ]
}

################################################################################
# Create CNAME Records
################################################################################

resource "aws_route53_record" "cname_records" {
  for_each = {
    for r in local.cname_matrix_expanded : r.key => r
  }

  zone_id = each.value.zone_id
  name    = each.value.name
  type    = each.value.type
  ttl     = 300
  records = [each.value.target]
}
