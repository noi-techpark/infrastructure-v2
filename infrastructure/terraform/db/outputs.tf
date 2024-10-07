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