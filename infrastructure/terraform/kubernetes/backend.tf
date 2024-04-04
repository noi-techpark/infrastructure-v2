################################################################################
## This file contains the configuration of Terraform Cloud and its providers.
################################################################################

terraform {
  cloud {
    organization = "noi-digital"

    workspaces {
      tags = ["opendatahub-kubernetes"]
    }
  }

  required_providers {
    # The configuration of the AWS provider and its required version.
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.43"
    }
    random = {
      source = "hashicorp/random"
      version = "~>3.6"
    }    
    # The configuration of the kubernetes provider and its required version.
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
  }

  # The required version of Terraform itself.
  required_version = "~> 1.7"
}
