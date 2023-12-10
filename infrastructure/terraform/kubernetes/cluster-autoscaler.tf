################################################################################
## This file contains the policy configuration for the Autoscaler addon for EKS.
## https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws/README.md
################################################################################

resource "aws_iam_policy" "cluster_autoscaler" {
  name_prefix = "cluster-autoscaler"
  description = "EKS cluster autoscaler policy for cluster ${data.aws_eks_cluster.default.id}"
  policy      = file("${path.module}/cluster-autoscaler-policy.json")
}
