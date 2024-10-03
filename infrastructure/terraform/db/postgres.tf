# Postgresql database internals like schemas, users, grants etc.

# provider "postgresql" {
#   scheme          = "awspostgres"
#   alias           = "pg-timeseries"
#   host            = aws_db_instance.pg-timeseries.address
#   username        = aws_db_instance.pg-timeseries.username
#   password        = aws_db_instance.pg-timeseries.password
#   sslmode         = "require"
#   connect_timeout = 15
#   superuser       = false
# }

# resource "postgresql_database" "bdp" {
#   provider = postgresql.pg-timeseries
#   name     = "bdp"
# }

# resource "postgresql_extension" "postgis" {
#   provider = postgresql.pg-timeseries
#   database = postgresql_database.bdp.name
#   name     = "postgis"
# }

# resource "random_password" "pg-timeseries-db-bdp" {
#   length  = 20
#   upper   = true
#   numeric = true
#   special = false
# }
# resource "random_password" "pg-timeseries-db-bdp-readonly" {
#   length  = 20
#   upper   = true
#   numeric = true
#   special = false
# }

# resource "postgresql_role" "bdp" {
#   provider = postgresql.pg-timeseries
#   name     = "bdp"
#   login    = "true"
#   password = random_password.pg-timeseries-db-bdp.result
# }

# resource "postgresql_role" "bdp_readonly" {
#   provider = postgresql.pg-timeseries
#   name     = "bdp_readonly"
#   login    = "true"
#   password = random_password.pg-timeseries-db-bdp-readonly.result
# }

# resource "postgresql_schema" "intimev2" {
#   provider = postgresql.pg-timeseries
#   name     = "intimev2"
#   database = postgresql_database.bdp.name
#   owner    = postgresql_role.bdp.name
# }

# resource "postgresql_grant" "bdp_all" {
#   provider    = postgresql.pg-timeseries
#   database    = postgresql_database.bdp.name
#   role        = postgresql_role.bdp.name
#   schema      = postgresql_schema.intimev2.name
#   object_type = "schema"
#   privileges  = ["CREATE", "USAGE"]
# }

# resource "postgresql_grant" "bdp_readonly_schema" {
#   provider    = postgresql.pg-timeseries
#   database    = postgresql_database.bdp.name
#   role        = postgresql_role.bdp_readonly.name
#   schema      = postgresql_schema.intimev2.name
#   object_type = "schema"
#   privileges  = ["USAGE"]
# }
# resource "postgresql_grant" "bdp_readonly_tables" {
#   provider    = postgresql.pg-timeseries
#   database    = postgresql_database.bdp.name
#   role        = postgresql_role.bdp_readonly.name
#   schema      = postgresql_schema.intimev2.name
#   object_type = "table"
#   objects     = [] # empty means all
#   privileges  = ["SELECT"]
# }
# resource "postgresql_default_privileges" "bdp_readonly_tables" {
#   provider    = postgresql.pg-timeseries
#   database    = postgresql_database.bdp.name
#   schema      = postgresql_schema.intimev2.name
#   role        = postgresql_role.bdp_readonly.name
#   owner       = postgresql_role.bdp.name
#   object_type = "table"
#   privileges  = ["SELECT"]
# }
