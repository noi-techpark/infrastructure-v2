################################################################################
## This file contains the EKS cluster.
## https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/17.24.0
################################################################################

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 18.0"

  # ----------------------------------------------------------------------------
  # General
  # ----------------------------------------------------------------------------
  cluster_name    = "aws-main-eu-01"
  cluster_version = "1.22"

  # ----------------------------------------------------------------------------
  # Network
  # ----------------------------------------------------------------------------
  vpc_id  = module.vpc.vpc_id
  subnets = module.vpc.private_subnets

  # ----------------------------------------------------------------------------
  # Authentication - this has to be managed manually.
  # ----------------------------------------------------------------------------
  manage_aws_auth_configmap = true

  aws_auth_roles = []

  aws_auth_users = [
    # Terraform.
    {
      userarn  = "arn:aws:iam::755952719952:user/odhv2-terraform"
      username = "terraform"
      groups   = ["system:masters"]
    },
    # Animeshon.
    {
      userarn  = "arn:aws:iam::755952719952:user/animeshon"
      username = "animeshon"
      groups   = ["system:masters"]
    },
    # Simon Dalvai.
    {
      userarn  = "arn:aws:iam::755952719952:user/sdalvai"
      username = "simon-dalvai"
      groups   = ["system:masters"]
    },
    # Rudi Thoeni.
    {
      userarn  = "arn:aws:iam::755952719952:user/Rudi"
      username = "rudi-thoeni"
      groups   = ["system:masters"]
    },
  ]

  aws_auth_accounts = []

  # ----------------------------------------------------------------------------
  # Node Groups
  # ----------------------------------------------------------------------------
  self_managed_node_groups = {
    # Turn off the default node group.
    default_node_group = {}

    main = {
      name = "main-pool"

      # Node group autoscaling.
      max_size     = 3
      desired_size = 1

      # Node instances.
      instance_type = "t3.medium"
    }
  }

  # ----------------------------------------------------------------------------
  # Tags
  # ----------------------------------------------------------------------------
  tags = {
    Environment = "proof-of-concept"
    Terraform   = "true"
  }
}
