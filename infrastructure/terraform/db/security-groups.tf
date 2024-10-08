
data "aws_security_group" "default" {
  name   = "default"
  vpc_id = data.aws_vpc.k8s-vpc.id
}

resource "aws_security_group" "postgres-homeoffice" {
  name        = "postgres-homeoffice"
  description = "Access to postgres from homeoffice"
  vpc_id      = data.aws_vpc.k8s-vpc.id
}

resource "aws_security_group" "postgres-test" {
  name        = "postgres"
  description = "Access to postgres testing database"
  vpc_id      = data.aws_vpc.k8s-vpc.id

  dynamic "ingress" {
    for_each = concat(
      local.ip_blocks.docker_hosts_test, 
      local.ip_blocks.eks_test, 
      local.ip_blocks.noi_offices
    )
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
  ip_blocks = {
    eks_test = [
      { descr = "eks-development", ip = "52.18.124.50/32" },
    ]
    noi_offices = [
      { descr = "NOI offices", ip = "46.18.28.240/32" },
      { descr = "NOI offices", ip = "46.18.28.241/32" },
      { descr = "NOI offices", ip = "46.18.28.242/32" },
    ]
    docker_hosts_test = [
      { descr = "docker-01-test", ip = "34.246.48.94/32" },
      { descr = "docker-02-test", ip = "54.170.76.169/32" },
      { descr = "docker-02-test", ip = "34.253.90.105/32" },
    ]
  }
}
