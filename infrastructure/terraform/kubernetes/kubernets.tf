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

# Authentication.
locals {
  aws_auth_roles = []
  aws_auth_users = [
    # Terraform.
    {
      userarn  = "arn:aws:iam::828408288281:user/terraform-ingress"
      username = "terraform"
      groups   = ["system:masters"]
    },
    # Animeshon.
    {
      userarn  = "arn:aws:iam::828408288281:user/animeshon"
      username = "animeshon"
      groups   = ["system:masters"]
    },
    # Simon Dalvai.
    {
      userarn  = "arn:aws:iam::828408288281:user/s.dalvai"
      username = "s.dalvai"
      groups   = ["system:masters"]
    },
    # Rudolf Thoeni.
    {
      userarn  = "arn:aws:iam::828408288281:user/r.thoeni"
      username = "r.thoeni"
      groups   = ["system:masters"]
    },
    # Clemens Zagler.
    {
      userarn  = "arn:aws:iam::828408288281:user/c.zagler"
      username = "c.zagler"
      groups   = ["system:masters"]
    },
    # Stefano Seppi.
    {
      userarn  = "arn:aws:iam::828408288281:user/s.seppi"
      username = "s.seppi"
      groups   = ["system:masters"]
    },
    # Martin Rabanser.
    {
      userarn  = "arn:aws:iam::828408288281:user/m.rabanser"
      username = "m.rabanser"
      groups   = ["system:masters"]
    },
  ]

  aws_auth_accounts = []
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