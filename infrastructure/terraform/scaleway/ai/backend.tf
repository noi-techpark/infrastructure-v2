################################################################################
## This file contains the backend and provider configuration.
################################################################################

terraform {
  backend "local" {}

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "~> 2.49"
    }
  }
  required_version = ">= 1.4.0"
}
