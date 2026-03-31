################################################################################
## This file contains the EFS CSI driver configuration for EKS.
## - IAM role for the EFS CSI controller service account
## - EFS filesystem and mount targets (one per private subnet)
## - Security group allowing NFS traffic from the cluster nodes
################################################################################

locals {
  efs_csi_service_account_namespace = "kube-system"
  efs_csi_service_account_name      = "efs-csi-controller-sa"
}

# -----------------------------------------------------------------------------
# IAM Role for EFS CSI Driver (IRSA)
# -----------------------------------------------------------------------------
module "efs_csi_controller_role" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version = "~> 5.11"

  create_role                   = true
  role_name                     = "${data.aws_eks_cluster.default.id}-efs-csi-controller"
  provider_url                  = replace(data.aws_eks_cluster.default.identity[0].oidc[0].issuer, "https://", "")
  role_policy_arns              = [aws_iam_policy.efs_csi_controller.arn]
  oidc_fully_qualified_subjects = ["system:serviceaccount:${local.efs_csi_service_account_namespace}:${local.efs_csi_service_account_name}"]
}

resource "aws_iam_policy" "efs_csi_controller" {
  name_prefix = "efs-csi-controller"
  description = "EKS efs-csi-controller policy for cluster ${data.aws_eks_cluster.default.id}"
  policy      = file("${path.module}/aws-efs-csi-driver-policy.json")
}

# -----------------------------------------------------------------------------
# EFS Filesystem
# -----------------------------------------------------------------------------
resource "aws_efs_file_system" "this" {
  creation_token = "${data.aws_eks_cluster.default.id}-efs"
  encrypted      = true

  tags = {
    Name      = "${data.aws_eks_cluster.default.id}-efs"
    Terraform = "true"
  }
}

# -----------------------------------------------------------------------------
# Security Group — allow NFS (port 2049) from cluster nodes
# -----------------------------------------------------------------------------
resource "aws_security_group" "efs" {
  name_prefix = "efs-"
  description = "Allow NFS traffic from EKS nodes"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port       = 2049
    to_port         = 2049
    protocol        = "tcp"
    security_groups = [module.eks.cluster_primary_security_group_id]
  }

  tags = {
    Name      = "${data.aws_eks_cluster.default.id}-efs"
    Terraform = "true"
  }
}

# -----------------------------------------------------------------------------
# Mount Targets — one per private subnet (covers all AZs)
# -----------------------------------------------------------------------------
resource "aws_efs_mount_target" "this" {
  for_each = toset(module.vpc.private_subnets)

  file_system_id  = aws_efs_file_system.this.id
  subnet_id       = each.value
  security_groups = [aws_security_group.efs.id]
}
