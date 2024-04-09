###############################################################################
# Authorizations for services  (github etc.)
###############################################################################

# Register IAM user for github actions access
# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/eks_access_entry
resource "aws_eks_access_entry" "github-actions" {
  cluster_name = module.eks.cluster_name
  principal_arn = "arn:aws:iam::${local.account_id}:user/github-actions"
  user_name = "github-actions"
  kubernetes_groups = []
}
# Attach cluser admin authorization to IAM users
# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/eks_access_policy_association
resource "aws_eks_access_policy_association" "github-actions" {
  cluster_name  = aws_eks_access_entry.github-actions.cluster_name
  policy_arn = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSEditPolicy"
  principal_arn = aws_eks_access_entry.github-actions.principal_arn

  access_scope {
    type       = "namespace"
    namespaces = ["collector", "core"]
  }
}