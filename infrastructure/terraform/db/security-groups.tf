
data "aws_security_group" "default" {
  name   = "default"
  vpc_id = data.aws_vpc.k8s-vpc.id
}

resource "aws_security_group" "postgres-homeoffice" {
  name        = "postgres-homeoffice"
  description = "Access to postgres from homeoffice"
  vpc_id      = data.aws_vpc.k8s-vpc.id
}

resource "aws_security_group" "postgres" {
  name        = "postgres"
  description = "Access to postgres database"
  vpc_id      = data.aws_vpc.k8s-vpc.id

  dynamic "ingress" {
    for_each = local.postgres_whitelist
    content {
      description = ingress.value.descr
      to_port     = 5432
      from_port   = 5432
      protocol    = "tcp"
      cidr_blocks = [ingress.value.ip]
    }
  }
}

locals {
  prod = var.ENVIRONMENT == "prod"
  ip_blocks = {
    eks_test = [
      { descr = "eks-development-public", ip = "52.18.124.50/32" },
      { descr = "eks-development-internal", ip = "10.0.0.0/16" },
    ]
    eks_prod = [
      { descr = "eks-production-public", ip = "34.249.69.97/32" },
      { descr = "eks-production-internal", ip = "10.0.0.0/16" },
    ]
    noi_offices = [
      { descr = "NOI offices", ip = "46.18.28.240/32" },
      { descr = "NOI offices", ip = "46.18.28.241/32" },
      { descr = "NOI offices", ip = "46.18.28.242/32" },
      { descr = "NOI offices", ip = "46.18.28.23/32" },
    ]
    docker_hosts_test = [
      { descr = "docker-01-test", ip = "34.246.48.94/32" },
      { descr = "docker-02-test", ip = "54.170.76.169/32" },
      { descr = "docker-03-test", ip = "34.253.90.105/32" },
    ]
    docker_hosts_prod = [
      { descr = "docker-01-prod", ip = "54.217.157.43/32" },
      { descr = "docker-02-prod", ip = "52.30.55.91/32" },
      { descr = "docker-03-prod", ip = "54.217.49.219/32" },
      { descr = "docker-04-prod", ip = "54.74.196.120/32" },
      { descr = "docker-05-prod", ip = "52.214.190.58/32" },
    ]
    grafana = [{descr = "grafana", ip = "52.214.46.195/32"}]
  }
  ip_blocks_test = concat(
    local.ip_blocks.docker_hosts_test, 
    local.ip_blocks.eks_test, 
    local.ip_blocks.noi_offices
  )
  ip_blocks_prod = concat(
    local.ip_blocks.docker_hosts_prod, 
    local.ip_blocks.eks_prod, 
    local.ip_blocks.noi_offices,
    local.ip_blocks.grafana
  )
  postgres_whitelist = local.prod ? local.ip_blocks_prod : local.ip_blocks_test
}
