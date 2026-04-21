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

variable "ENVIRONMENT" {
  type    = string
  default = "dev"
}

variable "INFERENCE_DEPLOYMENT_COUNT" {
  type    = number
  default = 1
}

variable "INFERENCE_NODE_TYPE" {
  type    = string
  default = "L4"
}

variable "INFERENCE_MODEL_NAME" {
  type    = string
  default = "mistral/mistral-nemo-instruct-2407"
}
