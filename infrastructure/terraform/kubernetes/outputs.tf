output "kubernetes_cluster_name" {
  value = module.eks.cluster_name
}

output "kubernetes_node_groups" {
  value = keys(module.eks.eks_managed_node_groups)
}

output "efs_file_system_id" {
  value = aws_efs_file_system.this.id
}
