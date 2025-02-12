################################################################################
## This file contains the configuration of Terraform Cloud and its providers.
################################################################################

terraform {
  cloud {
    organization = "noi-digital"

    workspaces {
      tags = ["opendatahub-db"]
    }
  }

  required_providers {
    # The configuration of the AWS provider and its required version.
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.55"
    }
    random = {
      source  = "hashicorp/random"
      version = "~>3.5"
    }
    # postgresql = {
    #   source  = "cyrilgdn/postgresql"
    #   version = "1.23.0"
    # }
  }

  # The required version of Terraform itself.
  required_version = "~> 1.9"
}
