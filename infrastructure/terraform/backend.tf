################################################################################
## This file contains the configuration of Terraform Cloud and its providers.
################################################################################

terraform {
  cloud {
    organization = "noi-digital"

    workspaces {
      name = "opendatahub-v2"
    }
  }

  required_providers {
    # The configuration of the AWS provider and its required version.
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }

  # The required version of Terraform itself.
  required_version = "~> 1.0"
}

# The instance of the AWS provider.
provider "aws" {
  # The default region used to provision resources.
  region = "eu-west-1"

  # The access credentials passed as variables by Terraform Cloud.
  access_key = var.AWS_ACCESS_KEY_ID
  secret_key = var.AWS_SECRET_ACCESS_KEY
}
