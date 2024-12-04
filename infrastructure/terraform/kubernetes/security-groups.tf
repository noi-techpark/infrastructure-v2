
resource "aws_security_group" "private-services" {
  name        = "eks-private-services"
  description = "Access to private services in EKS for office and homeoffice"
  vpc_id      = module.vpc.vpc_id

  dynamic "ingress" {
    for_each = concat(
      local.ip_blocks.noi_offices
    )
    content {
      description = ingress.value.descr
      cidr_blocks = [ingress.value.ip]
      from_port = 0
      to_port = 0
      protocol = "-1"
    }
  }
  
  tags = {
    Name = "eks-private-services"
  }
}

locals {
  ip_blocks = {
    noi_offices = [
      { descr = "NOI offices", ip = "46.18.28.240/32" },
      { descr = "NOI offices", ip = "46.18.28.241/32" },
      { descr = "NOI offices", ip = "46.18.28.242/32" },
    ]
  }
}
