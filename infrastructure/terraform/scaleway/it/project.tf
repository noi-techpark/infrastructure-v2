################################################################################
## This file creates the Scaleway project for the IT department.
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/account_project
################################################################################

resource "scaleway_account_project" "it" {
  name        = "IT"
  description = "Project for the IT department"
}
