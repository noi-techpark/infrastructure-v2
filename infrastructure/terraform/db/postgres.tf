# Postgresql database internals like schemas, users, grants etc.

provider "postgresql" {
    scheme = "awspostgres"
    alias = "odh-postgres"
    host = aws_db_instance.odh-postgres.address
    port = aws_db_instance.odh-postgres.port
    username = aws_db_instance.odh-postgres.username
    password = aws_db_instance.odh-postgres.password
    sslmode = "require"
    connect_timeout = 15
    superuser = false
}

resource "postgresql_database" "bdp" {
    provider = postgresql.odh-postgres
    name = "bdp"
}

resource "postgresql_extension" "postgis" {
    provider = postgresql.odh-postgres
    database = postgresql_database.bdp.name
    name = "postgis"
}

resource "random_password" "odh-postgres-db-bdp" {
    length = 20
    upper = true
    numeric = true
    special = false
}
resource "random_password" "odh-postgres-db-bdp-readonly" {
    length = 20
    upper = true
    numeric = true
    special = false
}

resource "postgresql_role" "bdp" {
    provider = postgresql.odh-postgres
    name = "bdp"
    login = "true"
    password = random_password.odh-postgres-db-bdp.result 
}

resource "postgresql_role" "bdp_readonly" {
    provider = postgresql.odh-postgres
    name = "bdp_readonly"
    login = "true"
    password = random_password.odh-postgres-db-bdp-readonly.result 
}

resource "postgresql_schema" "intimev2" {
    provider = postgresql.odh-postgres
    name = "intimev2"
    database = postgresql_database.bdp.name
    owner = postgresql_role.bdp.name
}

resource "postgresql_grant" "bdp_all" {
    provider = postgresql.odh-postgres
    database = postgresql_database.bdp.name
    role = postgresql_role.bdp.name
    schema = postgresql_schema.intimev2.name
    object_type = "schema"
    privileges = ["CREATE", "USAGE"]
}

resource "postgresql_grant" "bdp_readonly_schema" {
    provider = postgresql.odh-postgres
    database = postgresql_database.bdp.name
    role = postgresql_role.bdp_readonly.name
    schema = postgresql_schema.intimev2.name
    object_type = "schema"
    privileges = ["USAGE"]
}
resource "postgresql_grant" "bdp_readonly_tables" {
    provider = postgresql.odh-postgres
    database = postgresql_database.bdp.name
    role = postgresql_role.bdp_readonly.name
    schema = postgresql_schema.intimev2.name
    object_type = "table"
    objects = [] # empty means all
    privileges = ["SELECT"]
}
