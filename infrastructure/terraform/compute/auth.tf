###############################################################################
# The following resources have been extracted from the official AWS EKS module:
# https://github.com/terraform-aws-modules/terraform-aws-eks/blob/v19.13.0/main.tf#L464-L569
###############################################################################

# Authentication.
locals {
  aws_auth_users = [
    {
      userarn  = "arn:aws:iam::828408288281:user/animeshon"
      username = "animeshon"
      groups   = []
    },
    # Simon Dalvai.
    {
      userarn  = "arn:aws:iam::828408288281:user/s.dalvai"
      username = "s.dalvai"
      groups   = []
    },
    # Rudolf Thoeni.
    {
      userarn  = "arn:aws:iam::828408288281:user/r.thoeni"
      username = "r.thoeni"
      groups   = []
    },
    # Clemens Zagler.
    {
      userarn  = "arn:aws:iam::828408288281:user/c.zagler"
      username = "c.zagler"
      groups   = []
    },
    # Stefano Seppi.
    {
      userarn  = "arn:aws:iam::828408288281:user/s.seppi"
      username = "s.seppi"
      groups   = []
    },
    # Martin Rabanser.
    {
      userarn  = "arn:aws:iam::828408288281:user/m.rabanser"
      username = "m.rabanser"
      groups   = []
    },
  ]
}


# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/eks_access_entry
resource "aws_eks_access_entry" "default" {
  for_each = {
    for index, u in local.aws_auth_users:
    u.username => u
  }

  cluster_name = module.eks.cluster_name
  principal_arn = each.value.userarn
  user_name = each.value.username
  kubernetes_groups = each.value.groups
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/eks_access_policy_association
resource "aws_eks_access_policy_association" "clusterAdmins" {
  for_each = aws_eks_access_entry.default

  cluster_name  = each.value.cluster_name
  policy_arn = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"
  principal_arn = each.value.principal_arn

  access_scope {
    type       = "cluster"
    namespaces = []
  }
}