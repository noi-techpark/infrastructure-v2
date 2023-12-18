################################################################################
## This file contains the configuration of Terraform Cloud and its providers.
################################################################################

terraform {
  cloud {
    organization = "noi-digital"

    workspaces {
      tags = ["opendatahub-dns"]
    }
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.55"
    }
  }
  required_version = ">= 1.4.0"
}

