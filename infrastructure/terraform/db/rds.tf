# RDS instances and related security / network rules
resource "aws_db_subnet_group" "pg-timeseries" {
  name       = "pg-timeseries"
  subnet_ids = data.aws_subnets.public.ids
}

resource "aws_db_instance" "pg-timeseries" {
  identifier             = "pg-timeseries"
  instance_class         = var.DB_INSTANCE_TYPE
  allocated_storage      = var.DB_INSTANCE_STORAGE_GB # for testing make this the size it needs after first dump
  engine                 = "postgres"
  engine_version         = "16"
  username               = "postgres"
  password               = random_password.pg-timeseries-db-password.result
  db_subnet_group_name   = aws_db_subnet_group.pg-timeseries.name
  publicly_accessible    = true
  skip_final_snapshot    = true
  vpc_security_group_ids = [
    data.aws_security_group.default.id, 
    aws_security_group.postgres.id, 
    aws_security_group.postgres-homeoffice.id,
    aws_security_group.postgres-dbmigration.id
  ]
  storage_type = "gp3"
  storage_throughput = var.DB_INSTANCE_STORAGE_THROUGHPUT
  iops = var.DB_INSTANCE_STORAGE_IOPS
  
  # Backups:
  backup_retention_period = 7
  backup_window = "02:00-04:30"
  copy_tags_to_snapshot = true
  
  # Maintenance:
  maintenance_window = "Thu:01:15-Thu:01:45"
  auto_minor_version_upgrade = true

  deletion_protection = true
  apply_immediately = true
}