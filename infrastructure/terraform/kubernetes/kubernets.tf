################################################################################
## This file contains the kubernetes provider.
## https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs
################################################################################

locals {
  cluster_name = data.tfe_outputs.compute.nonsensitive_values.kubernetes_cluster_name
}

data "aws_eks_cluster" "default" {
  name = local.cluster_name
}

data "aws_eks_cluster_auth" "default" {
  name = local.cluster_name
}

# The instance of the kubernetes provider.
provider "kubernetes" {
  host                   = data.aws_eks_cluster.default.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.default.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.default.token
}

# Addons.
 locals {
   cluster_addons = {
     coredns = {
       most_recent = true
     }
     kube-proxy = {
       most_recent = true
     }
     vpc-cni = {
       most_recent = true
       before_compute = true
       configuration_values = jsonencode({
         env = {
           # Reference docs https://docs.aws.amazon.com/eks/latest/userguide/cni-increase-ip-addresses.html
           ENABLE_PREFIX_DELEGATION = "true"
           WARM_PREFIX_TARGET       = "1"
         }
       })
     }
     aws-ebs-csi-driver = {
       most_recent              = true
       service_account_role_arn = "arn:aws:iam::${local.account_id}:role/${data.aws_eks_cluster.default.id}-ebs-csi-controller"
     }
   }
 }