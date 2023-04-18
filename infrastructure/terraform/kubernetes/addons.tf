###############################################################################
# The following resources have been extracted from the official AWS EKS module:
# https://github.com/terraform-aws-modules/terraform-aws-eks/blob/v19.13.0/main.tf#L378-L438
###############################################################################

resource "aws_eks_addon" "this" {
  depends_on = [
    kubernetes_config_map.aws_auth,
    kubernetes_config_map_v1_data.aws_auth,
  ]

  for_each = local.cluster_addons

  cluster_name = data.aws_eks_cluster.default.id
  addon_name   = try(each.value.name, each.key)

  addon_version            = try(each.value.addon_version, data.aws_eks_addon_version.this[each.key].version)
  configuration_values     = try(each.value.configuration_values, null)
  preserve                 = try(each.value.preserve, null)
  resolve_conflicts        = try(each.value.resolve_conflicts, "OVERWRITE")
  service_account_role_arn = try(each.value.service_account_role_arn, null)

  timeouts {
    create = try(each.value.timeouts.create, null)
    update = try(each.value.timeouts.update, null)
    delete = try(each.value.timeouts.delete, null)
  }
}

data "aws_eks_addon_version" "this" {
  depends_on = [
    kubernetes_config_map.aws_auth,
    kubernetes_config_map_v1_data.aws_auth,
  ]

  for_each = local.cluster_addons

  addon_name         = try(each.value.name, each.key)
  kubernetes_version = data.aws_eks_cluster.default.version
  most_recent        = try(each.value.most_recent, null)
}
