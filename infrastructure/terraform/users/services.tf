# This file creates users for services

# User for github actions ci/cd access
resource "aws_iam_user" "github-actions" {
  name = "github-actions"
}

resource "aws_iam_user_policy_attachment" "github_actions_update_kubeconfig" {
    user = aws_iam_user.github-actions.name
    policy_arn = aws_iam_policy.eks_update_kubeconfig.arn
}

# Needed to use update-kubeconfig, login for kubectl
# https://docs.aws.amazon.com/eks/latest/userguide/security_iam_id-based-policy-examples.html
resource "aws_iam_policy" "eks_update_kubeconfig" {
    name = "eks-update-kubeconfig"
    description = "Needed for IAM users to be able to log in kubectl via AWS CLI"
    policy = data.aws_iam_policy_document.eks_update_kubeconfig.json
}

data "aws_iam_policy_document" "eks_update_kubeconfig" {
  statement {
    sid       = ""
    effect    = "Allow"
    resources = ["*"]

    actions = [
      "eks:DescribeCluster",
      "eks:ListClusters",
    ]
  }
}
