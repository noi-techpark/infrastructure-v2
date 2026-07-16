################################################################################
## This file contains the variables provided by Terraform Cloud.
## https://www.terraform.io/language/values/variables
################################################################################

variable "SCW_ACCESS_KEY" {
  type    = string
  default = ""
}

variable "SCW_SECRET_KEY" {
  type      = string
  default   = ""
  sensitive = true
}

variable "SCW_PROJECT_ID" {
  type    = string
  default = ""
}

variable "SCW_ORGANIZATION_ID" {
  type    = string
  default = ""
}
