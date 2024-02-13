################################################################################
## This file contains the configuration of Terraform Cloud and its providers.
################################################################################

terraform {
  cloud {
    organization = "noi-digital"

    workspaces {
      tags = ["opendatahub-kubernetes-managed"]
    } 
  }

  required_providers {
    # The configuration of the AWS provider and its required version.
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.36"
    }
  }

  # The required version of Terraform itself.
  required_version = "~> 1.7"
}
