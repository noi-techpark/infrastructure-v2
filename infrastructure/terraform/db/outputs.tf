# ODH postgres instance coordinates

output "odh_postgres_hostname" {
  description = "ODH postgres instance hostname"
  value       = aws_db_instance.pg-timeseries.address
}
output "odh_postgres_port" {
  description = "ODH postgres instance port"
  value       = aws_db_instance.pg-timeseries.port
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
output "odh_postgres_password_bdp" {
  description = "ODH postgres instance password for bdp user"
  value       = random_password.pg-timeseries-db-bdp.result
  sensitive   = true
}
output "odh_postgres_password_bdp_readonly" {
  description = "ODH postgres instance password for bdp readonly user"
  value       = random_password.pg-timeseries-db-bdp-readonly.result
  sensitive   = true
}
output "postgres_homeoffice_sg_id" {
  description = "Security group ID for postgres access from home"
  value       = aws_security_group.postgres-homeoffice.id
}