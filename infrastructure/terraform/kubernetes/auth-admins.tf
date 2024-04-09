###############################################################################
# Authorizations for personal IAM users 
###############################################################################

locals {
  aws_auth_admins = [
    for user in local.eks_admins: 
    {
      userarn  = "arn:aws:iam::${local.account_id}:user/${user}"
      username = user
      groups   = []
    }
  ]
}

# Register IAM users as kubernetes users
# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/eks_access_entry
resource "aws_eks_access_entry" "default" {
  for_each = {
    for index, u in local.aws_auth_admins:
    u.username => u
  }

  cluster_name = module.eks.cluster_name
  principal_arn = each.value.userarn
  user_name = each.value.username
  kubernetes_groups = each.value.groups
}

# Attach cluser admin authorization to IAM users
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