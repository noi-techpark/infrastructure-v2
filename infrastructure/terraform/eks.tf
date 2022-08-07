################################################################################
## This file contains the EKS cluster.
## https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/18.26.6
################################################################################

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 18.26"

  # ----------------------------------------------------------------------------
  # General
  # ----------------------------------------------------------------------------
  cluster_name    = "aws-main-eu-01"
  cluster_version = "1.22"

  # ----------------------------------------------------------------------------
  # Network
  # ----------------------------------------------------------------------------
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # ----------------------------------------------------------------------------
  # Logging
  # ----------------------------------------------------------------------------
  create_cloudwatch_log_group = false # We are missing IAM permission.

  # ----------------------------------------------------------------------------
  # Authentication - this has to be managed manually.
  # ----------------------------------------------------------------------------
  create_aws_auth_configmap = true
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
    # Rudolf Thoeni.
    {
      userarn  = "arn:aws:iam::755952719952:user/Rudi"
      username = "rudolf-thoeni"
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
  # Security Groups
  # ----------------------------------------------------------------------------
  node_security_group_additional_rules = {
    ingress_allow_access_from_control_plane = {
      type                          = "ingress"
      protocol                      = "tcp"
      from_port                     = 9443
      to_port                       = 9443
      source_cluster_security_group = true
      description                   = "Allow access from control plane to webhook port of AWS load balancer controller"
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
