resource "random_password" "odh-postgres-db-password" {
    length = 20
    upper = true
    numeric = true
    special = false
}

resource "aws_db_subnet_group" "odh-postgres" {
    name = "odh-postgres"
    subnet_ids = module.vpc.public_subnets
}

resource "aws_db_instance" "odh-postgres" {
    identifier = "odh-postgres"
    instance_class = "db.t4g.medium"
    allocated_storage = 20
    engine = "postgres"
    engine_version = "15.3"
    username = "postgres"
    password = random_password.odh-postgres-db-password.result
    db_subnet_group_name = aws_db_subnet_group.odh-postgres.name
    publicly_accessible = true
    skip_final_snapshot = true
}
