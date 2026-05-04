################################################################################
## This file contains the backend and provider configuration.
################################################################################

terraform {
  backend "s3" {
    key                         = "users/terraform.tfstate"
    region                      = "fr-par"
    endpoints = {
      s3 = "https://s3.fr-par.scw.cloud"
    }
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    bucket                      = "tfstate-42"
  }

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "~> 2.49"
    }
  }
  required_version = ">= 1.4.0"
}
