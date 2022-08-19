################################################################################
## This file contains the EKS cluster.
## https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/18.26.6
################################################################################

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 18.27"

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
    # # Simon Dalvai.
    # {
    #   userarn  = "arn:aws:iam::463112166163:user/sdalvai"
    #   username = "simon-dalvai"
    #   groups   = ["system:masters"]
    # },
    # # Rudolf Thoeni.
    # {
    #   userarn  = "arn:aws:iam::463112166163:user/Rudi"
    #   username = "rudolf-thoeni"
    #   groups   = ["system:masters"]
    # },
  ]

  aws_auth_accounts = []

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
    # Turn off the default node group.
    default_node_group = {}

    main = {
      name = "main-pool"

      # Node group autoscaling.
      max_size     = 1
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

    ingress_self_all = {
      description = "Node to node all ports/protocols"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      type        = "ingress"
      self        = true
    }

    egress_all = {
      description      = "Node all egress"
      protocol         = "-1"
      from_port        = 0
      to_port          = 0
      type             = "egress"
      cidr_blocks      = ["0.0.0.0/0"]
      ipv6_cidr_blocks = ["::/0"]
    }
  }

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

  # ----------------------------------------------------------------------------
  # Tags
  # ----------------------------------------------------------------------------
  tags = {
    Environment = "proof-of-concept"
    Terraform   = "true"
  }
}
