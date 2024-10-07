# ODH postgres instance coordinates

output "odh_postgres_password_bdp" {
  description = "ODH postgres instance password for bdp user"
  value       = postgresql_role.bdp.password
  sensitive   = true
}
output "odh_postgres_password_bdp_readonly" {
  description = "ODH postgres instance password for bdp readonly user"
  value       = postgresql_role.bdp_readonly.password
  sensitive   = true
}