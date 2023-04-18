###############################################################################
# The following resources have been extracted from the official AWS EKS module:
# https://github.com/terraform-aws-modules/terraform-aws-eks/blob/v19.13.0/main.tf#L464-L569
###############################################################################

data "aws_iam_role" "default" {
  for_each = toset(local.node_groups)
  name     = "${each.key}-pool-node-group"
}

locals {
  aws_auth_configmap_data = {
    mapRoles = yamlencode(concat(
      [for role in data.aws_iam_role.default : {
        rolearn  = role.arn
        username = "system:node:{{EC2PrivateDNSName}}"
        groups = [
          "system:bootstrappers",
          "system:nodes",
        ]
        }
      ],
      local.aws_auth_roles
    ))
    mapUsers    = yamlencode(local.aws_auth_users)
    mapAccounts = yamlencode(local.aws_auth_accounts)
  }
}

resource "kubernetes_config_map" "aws_auth" {
  metadata {
    name      = "aws-auth"
    namespace = "kube-system"
  }

  data = local.aws_auth_configmap_data

  lifecycle {
    ignore_changes = [data, metadata[0].labels, metadata[0].annotations]
  }
}

resource "kubernetes_config_map_v1_data" "aws_auth" {
  depends_on = [kubernetes_config_map.aws_auth]

  force = true

  metadata {
    name      = "aws-auth"
    namespace = "kube-system"
  }

  data = local.aws_auth_configmap_data
}
