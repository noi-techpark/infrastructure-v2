################################################################################
## This file contains the veriables provided by Terraform Cloud.
## https://www.terraform.io/language/values/variables
################################################################################

# The environment variables used to access AWS through a service account.
variable "AWS_ACCESS_KEY_ID" {
  type    = string
  default = ""
}

variable "AWS_SECRET_ACCESS_KEY" {
  type    = string
  default = ""
}

variable "ENVIRONMENT" {
  type    = string
  default = "dev"
}

