################################################################################
## This file contains the configuration of Terraform Cloud and its providers.
################################################################################

terraform {
  cloud {
    organization = "noi-digital"

    workspaces {
      tags = ["opendatahub-postgres-timeseries"]
    }
  }

  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~>3.5"
    }
    postgresql = {
      source  = "cyrilgdn/postgresql"
      version = "1.23.0"
    }
  }

  # The required version of Terraform itself.
  required_version = "~> 1.9"
}
