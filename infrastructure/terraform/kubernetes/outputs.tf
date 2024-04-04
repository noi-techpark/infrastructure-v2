output "kubernetes_cluster_name" {
  value = module.eks.cluster_name
}

output "kubernetes_node_groups" {
  value = keys(module.eks.eks_managed_node_groups)
}
