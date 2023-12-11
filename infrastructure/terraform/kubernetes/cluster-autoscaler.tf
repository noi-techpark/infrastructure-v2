################################################################################
## This file contains the policy configuration for the Autoscaler addon for EKS.
## https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws/README.md
################################################################################

locals {
  cluster_autscaler_service_account_namespace = "kube-system"
  cluster_autscaler_service_account_name      = "aws-cluster-autoscaler"
}

module "cluster_autscaler_role" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version = "~> 5.11"

  create_role                   = true
  role_name                     = "${data.aws_eks_cluster.default.id}-cluster-autoscaler"
  provider_url                  = replace(data.aws_eks_cluster.default.identity[0].oidc[0].issuer, "https://", "")
  role_policy_arns              = [aws_iam_policy.cluster_autoscaler.arn]
  oidc_fully_qualified_subjects = ["system:serviceaccount:${local.cluster_autscaler_service_account_namespace}:${local.cluster_autscaler_service_account_name}"]
}

resource "aws_iam_policy" "cluster_autoscaler" {
  name_prefix = "cluster-autoscaler"
  description = "EKS cluster autoscaler policy for cluster ${data.aws_eks_cluster.default.id}"
  policy      = file("${path.module}/cluster-autoscaler-policy.json")
}
