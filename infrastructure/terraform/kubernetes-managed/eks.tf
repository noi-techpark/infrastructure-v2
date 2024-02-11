module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "20.2.1"

  # ----------------------------------------------------------------------------
  # General
  # ----------------------------------------------------------------------------
  cluster_name    = "aws-main-eu-02"
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
  # Cluster Permissions - add the current caller identity as an administrator.
  # ----------------------------------------------------------------------------
  enable_cluster_creator_admin_permissions = true

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
      most_recent    = true
      before_compute = true
      configuration_values = jsonencode({
        env = {
          ENABLE_PREFIX_DELEGATION = "true"
          WARM_PREFIX_TARGET       = "1"
        }
      })
    }
  }

  # ----------------------------------------------------------------------------
  # Node Groups
  # ----------------------------------------------------------------------------
  eks_managed_node_groups = {
    main = {
      name = "main-pool"

      # Node group autoscaling.
      max_size     = 5
      desired_size = 3
      min_size     = 3 # NOTE: the minimum size must be at least equal to the amount of subnets (zones).

      # Node instances.
      instance_type = "t3.medium"

      use_custom_launch_template = false

      ami_type = "BOTTLEROCKET_x86_64"
      platform = "bottlerocket"
    }
  }

  # ----------------------------------------------------------------------------
  # Authentication & Authorization
  #
  # https://aws.amazon.com/blogs/containers/a-deep-dive-into-simplified-amazon-eks-access-management-controls/
  # https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles
  # ----------------------------------------------------------------------------
  access_entries = {
    # Animeshon.
    animeshon = {
      kubernetes_groups = []
      principal_arn     = "arn:aws:iam::828408288281:user/animeshon"

      policy_associations = {
        default = {
          # Values: [AmazonEKSClusterAdminPolicy, AmazonEKSAdminPolicy, AmazonEKSEditPolicy, AmazonEKSViewPolicy].
          policy_arn = "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy"
          access_scope = {
            # Values: [cluster, namespace].
            type       = "cluster",
            namespaces = []
          }
        }
      }
    }

    # NOTE: the Terraform Access Entry has already been created by the module.
    # See `enable_cluster_creator_admin_permissions` option for more information.
  }
}
