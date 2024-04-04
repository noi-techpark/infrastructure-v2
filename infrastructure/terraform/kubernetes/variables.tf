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

# Some resources require the cluster to already be running. 
# if initial_create is set "true", those resources are skipped
# launch the very firstapply with -var "INITIAL_CREATE=true" and then apply again
variable "INITIAL_CREATE" {
  type = string
  default = "false"
}