# RDS instances and related security / network rules

resource "random_password" "pg-timeseries-db-password" {
  length  = 20
  upper   = true
  numeric = true
  special = false
}

resource "aws_db_subnet_group" "pg-timeseries" {
  name       = "pg-timeseries"
  subnet_ids = data.aws_subnets.public.ids
}

resource "aws_db_instance" "pg-timeseries" {
  identifier             = "pg-timeseries"
  instance_class         = var.DB_INSTANCE_TYPE
  allocated_storage      = var.DB_INSTANCE_STORAGE_GB
  engine                 = "postgres"
  engine_version         = "16"
  username               = "postgres"
  password               = random_password.pg-timeseries-db-password.result
  db_subnet_group_name   = aws_db_subnet_group.pg-timeseries.name
  publicly_accessible    = true
  skip_final_snapshot    = true
  vpc_security_group_ids = [
    data.aws_security_group.default.id, 
    aws_security_group.postgres-test.id, 
  ]
}