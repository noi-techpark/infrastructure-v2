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
      version = "~> 4.55"
    }

    # The configuration of the kubernetes provider and its required version.
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.18"
    }
  }

  # The required version of Terraform itself.
  required_version = "~> 1.4"
}

data "tfe_outputs" "compute" {
  organization = "noi-digital"
  workspace    = var.TF_COMPUTE_WORKSPACE
}
