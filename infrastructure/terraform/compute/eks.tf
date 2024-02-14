################################################################################
## This file contains the EKS cluster.
## https://registry.terraform.io/modules/terraform-aws-modules/eks/aws/19.21.0
################################################################################

resource  "aws_key_pair" "kubernetes-node" {
  key_name = "ec2-kubernetes-node"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCvyTatYJ7RURFzXBvFoLRpIWMuSXFniLFgkwtmnPV+Tk0MM0DCw12a2EHr3kptGrRyLEDP5agbwgFJEcCZJQNlvdRmaau2wqxyxtu9wML8FO05xDlYn6PPI1CKaFZU/EDwR7nNvcRuUEAg6TftDJ6tuZUxqEVId8zEVX1+0WgaoxvfFE/81erSPqUEDMzce1ENbrNnJltGSGEoEnmNg3F3EHjwyX+HQZAhJyw+AM1NWYc0kVFQF/kKp+k49ts0fBy3VQOAjqXNV4IbPDhFMCcT1GA6aMYjzJs3r7Y9BlZqMhlRc6CdoQZwiUR1PjKmUhn5zcUaySDdyB4GkmCOsi4kPGC4vOQ9+4mkCP8ZfFeFRnraDzQtoJ0MMK/Y1rXoOwKupjLF0+DfR2gtcTBivtE+By722Gb8Ttc6mKVEG5L/G8Je7yvLQUQgDLezA3G1KnW4BsNuZ2rG8my+5gbUoC6eoNmy6QKg8nYsdx+eJdjVZ4DvkxIOg5aYYxS7H0AF5JM= digital@noi.bz.it"
}

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
      max_size     = 5
      desired_size = 3
      min_size     = 3 # NOTE: the minimum size must be at least equal to the amount of subnets (zones). 
      key_name = aws_key_pair.kubernetes-node.key_name

      # Node instances.
      instance_type = var.EKS_MAIN_POOL_INSTANCE_TYPE
      
      # enable more pods per node using IP prefix: https://docs.aws.amazon.com/eks/latest/userguide/cni-increase-ip-addresses.html
      bootstrap_extra_args = "--use-max-pods false --kubelet-extra-args '--max-pods=110'"

      # IAM roles.
      iam_role_use_name_prefix = false
    }
  }

  eks_managed_node_groups = {
    main = {
      name = "main-pool-managed"

      # Node group autoscaling.
      max_size     = 1
      desired_size = 1
      min_size     = 1 # NOTE: the minimum size must be at least equal to the amount of subnets (zones).
      key_name = aws_key_pair.kubernetes-node.key_name

      # Node instances.
      instance_type = "t3.medium"

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
