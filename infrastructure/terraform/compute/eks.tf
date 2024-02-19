################################################################################
## This file contains the EKS cluster.
## https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/19.21.0
################################################################################

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.2"

  # ----------------------------------------------------------------------------
  # General
  # ----------------------------------------------------------------------------
  cluster_name    = "aws-main-eu-01"
  cluster_version = "1.29"

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
  # Cluster Permissions - add the current caller identity as an administrator.
  # ----------------------------------------------------------------------------
  enable_cluster_creator_admin_permissions = true

  eks_managed_node_groups = {
    main = {
      name = "main-pool-deprecated"

      # Node group autoscaling.
      min_size     = 1 # NOTE: the minimum size must be at least equal to the amount of subnets (zones).
      max_size     = 5
      desired_size = 3

      # Node instances.
      instance_types = [var.EKS_MAIN_POOL_INSTANCE_TYPE]

      use_custom_launch_template = false

      ami_type = "BOTTLEROCKET_x86_64"
      platform = "bottlerocket"

      # IAM roles.
      iam_role_use_name_prefix = false
    }
  }

  # ----------------------------------------------------------------------------
  # Security Groups
  # ----------------------------------------------------------------------------
  # https://github.com/terraform-aws-modules/terraform-aws-eks/blob/master/docs/network_connectivity.md#security-groups
  cluster_security_group_additional_rules = {
    egress_nodes_ephemeral_ports_tcp = {
      description                = "To node 1025-65535"
      protocol                   = "tcp"
      from_port                  = 1025
      to_port                    = 65535
      type                       = "egress"
      source_node_security_group = true
    }
  }

  # Extend node-to-node security group rules
  node_security_group_additional_rules = {
    ingress_self_all = {
      description = "Node to node all ports/protocols"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      type        = "ingress"
      self        = true
    }
  }

  # ----------------------------------------------------------------------------
  # Tags
  # ----------------------------------------------------------------------------
  tags = {
    Terraform = "true"
  }
}
