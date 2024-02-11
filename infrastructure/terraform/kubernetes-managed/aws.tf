################################################################################
## This file contains the AWS provider.
## https://registry.terraform.io/providers/hashicorp/aws/latest/docs
################################################################################

# The instance of the AWS provider.
provider "aws" {
  # The default region used to provision resources.
  region = "eu-west-1"

  # The access credentials passed as variables by Terraform Cloud.
  access_key = var.AWS_ACCESS_KEY_ID
  secret_key = var.AWS_SECRET_ACCESS_KEY
}

data "aws_caller_identity" "current" {}
