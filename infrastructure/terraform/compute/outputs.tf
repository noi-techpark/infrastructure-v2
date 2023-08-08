output "kubernetes_cluster_name" {
  value = module.eks.cluster_name
}

output "kubernetes_node_groups" {
  value = keys(module.eks.self_managed_node_groups)
}

# ODH postgres instance coordinates
output "odh_postgres_hostname" {
  description = "ODH postgres instance hostname"
  value = aws_db_instance.odh-postgres.address
  sensitive = true
}
output "odh_postgres_port" {
  description = "ODH postgres instance port"
  value = aws_db_instance.odh-postgres.port
  sensitive = true
}
output "odh_postgres_username" {
  description = "ODH postgres instance root username"
  value = aws_db_instance.odh-postgres.username
  sensitive = true
}
output "odh_postgres_password" {
  description = "ODH postgres instance root password"
  value = aws_db_instance.odh-postgres.password
  sensitive = true
}