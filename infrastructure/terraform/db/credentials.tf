resource "random_password" "pg-timeseries-db-password" {
  length  = 20
  upper   = true
  numeric = true
  special = false
}

resource "random_password" "pg-timeseries-db-bdp" {
  length  = 20
  upper   = true
  numeric = true
  special = false
}
resource "random_password" "pg-timeseries-db-bdp-readonly" {
  length  = 20
  upper   = true
  numeric = true
  special = false
}