# RDS instances and related security / network rules

resource "random_password" "odh-postgres-db-password" {
    length = 20
    upper = true
    numeric = true
    special = false
}

resource "aws_db_subnet_group" "odh-postgres" {
    name = "odh-postgres"
    subnet_ids = data.aws_subnets.public.ids
}

resource "aws_security_group" "allow-postgres-noi" {
    name = "allow-postgres-noi"
    description = "allow postgres RDS access to users in NOI subnet"
    vpc_id = data.aws_vpc.k8s-vpc.id
    
    ingress {
        description = "Postgres from NOI"
        to_port = 5432
        from_port = 0
        protocol = "tcp"
        cidr_blocks = ["46.18.27.34/32"]
    }
}
resource "aws_security_group" "allow-postgres-all" {
    name = "allow-postgres-all"
    description = "allow postgres RDS access to everyone"
    vpc_id = data.aws_vpc.k8s-vpc.id

    
    ingress {
        description = "Postgres from Everywhere"
        to_port = 5432
        from_port = 0
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

data "aws_security_group" "default" {
    name = "default"
    vpc_id = data.aws_vpc.k8s-vpc.id
}

resource "aws_db_instance" "odh-postgres" {
    identifier = "odh-postgres"
    instance_class = var.DB_INSTANCE_TYPE
    allocated_storage = 100 
    engine = "postgres"
    engine_version = "15.3"
    username = "postgres"
    password = random_password.odh-postgres-db-password.result
    db_subnet_group_name = aws_db_subnet_group.odh-postgres.name
    publicly_accessible = true
    skip_final_snapshot = true
    vpc_security_group_ids = [data.aws_security_group.default.id, aws_security_group.allow-postgres-noi.id, aws_security_group.allow-postgres-all.id]
}