################################################################################
## This file contains the Scaleway provider.
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs
################################################################################

provider "scaleway" {
  access_key      = var.SCW_ACCESS_KEY
  secret_key      = var.SCW_SECRET_KEY
  project_id      = var.SCW_PROJECT_ID
  organization_id = var.SCW_ORGANIZATION_ID
  region          = "fr-par"
  zone            = "fr-par-1"
}
