################################################################################
## This file contains the kubernetes provider.
## https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs
################################################################################

locals {
  cluster_name = data.tfe_outputs.compute.nonsensitive_values.kubernetes_cluster_name
  node_groups  = data.tfe_outputs.compute.nonsensitive_values.kubernetes_node_groups
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
      userarn  = "arn:aws:iam::463112166163:user/odh-v2-terraform"
      username = "terraform"
      groups   = ["system:masters"]
    },
    # Animeshon.
    {
      userarn  = "arn:aws:iam::463112166163:user/animeshon"
      username = "animeshon"
      groups   = ["system:masters"]
    },
    # Simon Dalvai.
    {
      userarn  = "arn:aws:iam::463112166163:user/s.dalvai-dev-cli"
      username = "s.dalvai-dev-cli"
      groups   = ["system:masters"]
    },
    # Rudolf Thoeni.
    {
      userarn  = "arn:aws:iam::463112166163:user/r.thoeni-dev-cli"
      username = "r.thoeni-dev-cli"
      groups   = ["system:masters"]
    },
    # Clemens Zagler.
    {
      userarn  = "arn:aws:iam::463112166163:user/c.zagler-dev-cli"
      username = "c.zagler-dev-cli"
      groups   = ["system:masters"]
    },
    # Stefano Seppi.
    {
      userarn  = "arn:aws:iam::463112166163:user/s.seppi-dev-cli"
      username = "s.seppi-dev-cli"
      groups   = ["system:masters"]
    },
    # Martin Rabanser.
    {
      userarn  = "arn:aws:iam::463112166163:user/m.rabanser-dev-cli"
      username = "m.rabanser-dev-cli"
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
    }
    aws-ebs-csi-driver = {
      most_recent              = true
      service_account_role_arn = "arn:aws:iam::${local.account_id}:role/${data.aws_eks_cluster.default.id}-ebs-csi-controller"
    }
  }
}
