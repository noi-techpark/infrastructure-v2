################################################################################
## This file contains the EKS cluster.
## https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/19.10.0
################################################################################

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 19.10"

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
  # Addons
  # ----------------------------------------------------------------------------
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
      most_recent = true
      service_account_role_arn = "arn:aws:iam::${local.account_id}:role/${module.eks.cluster_name}-ebs-csi-controller"
    }
  }

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
  node_security_group_additional_rules    = {}
  cluster_security_group_additional_rules = {}

  # ----------------------------------------------------------------------------
  # Tags
  # ----------------------------------------------------------------------------
  tags = {
    Environment = "proof-of-concept"
    Terraform   = "true"
  }
}
