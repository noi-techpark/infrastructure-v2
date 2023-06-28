################################################################################
## This file contains the EKS cluster.
## https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/19.13.0
################################################################################

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 19.13"

  # ----------------------------------------------------------------------------
  # General
  # ----------------------------------------------------------------------------
  cluster_name    = "aws-main-eu-01"
  cluster_version = "1.24"

  # ----------------------------------------------------------------------------
  # Control Plane Access
  # ----------------------------------------------------------------------------
  cluster_endpoint_public_access = true

  # ----------------------------------------------------------------------------
  # Network
  # ----------------------------------------------------------------------------
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # ----------------------------------------------------------------------------
  # Addons - managed in the kubernetes workspace (see addons.tf).
  # ----------------------------------------------------------------------------
  cluster_addons = {}

  # ----------------------------------------------------------------------------
  # Authentication - managed in the kubernetes workspace (see auth.tf).
  # ----------------------------------------------------------------------------
  create_aws_auth_configmap = false
  manage_aws_auth_configmap = false

  # ----------------------------------------------------------------------------
  # Node Groups
  # ----------------------------------------------------------------------------
  self_managed_node_group_defaults = {
    # Enable discovery of autoscaling groups by cluster-autoscaler.
    autoscaling_group_tags = {
      "k8s.io/cluster-autoscaler/enabled" : true,
      "k8s.io/cluster-autoscaler/aws-main-eu-01" : "owned",
    }
  }

  self_managed_node_groups = {
    main = {
      name = "main-pool"

      # Node group autoscaling.
      max_size     = 1
      desired_size = 1

      # Node instances.
      instance_type = "t3.medium"

      # IAM roles.
      iam_role_use_name_prefix = false
    }
  }

  # ----------------------------------------------------------------------------
  # Security Groups
  # ----------------------------------------------------------------------------
  node_security_group_additional_rules    = {}
  cluster_security_group_additional_rules = {}

  # ----------------------------------------------------------------------------
  # Tags
  # ----------------------------------------------------------------------------
  tags = {
    Terraform = "true"
  }
}
