# ODH postgres instance coordinates

output "odh_postgres_hostname" {
  description = "ODH postgres instance hostname"
  value       = aws_db_instance.pg-timeseries.address
  sensitive   = true
}
output "odh_postgres_port" {
  description = "ODH postgres instance port"
  value       = aws_db_instance.pg-timeseries.port
  sensitive   = true
}
output "odh_postgres_username" {
  description = "ODH postgres instance root username"
  value       = aws_db_instance.pg-timeseries.username
  sensitive   = true
}
output "odh_postgres_password" {
  description = "ODH postgres instance root password"
  value       = aws_db_instance.pg-timeseries.password
  sensitive   = true
}
# output "odh_postgres_password_bdp" {
#   description = "ODH postgres instance password for bdp user"
#   value       = postgresql_role.bdp.password
#   sensitive   = true
# }
# output "odh_postgres_password_bdp_readonly" {
#   description = "ODH postgres instance password for bdp readonly user"
#   value       = postgresql_role.bdp_readonly.password
#   sensitive   = true
# }