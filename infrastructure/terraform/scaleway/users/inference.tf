################################################################################
## This file manages Scaleway IAM applications, policies, and API keys for
## generative AI / inference access.
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/iam_application
##
## To add a new application, add an entry to local.inference_applications.
## To add a new project, add an entry to local.inference_projects and reference
## it by key in the application's `project` field.
##
################################################################################

locals {
  # Add a new entry here if an application needs access to a different project.
  inference_projects = {
    default = "Default"
  }

  # Add a new entry here to create a new IAM application with its API key.
  inference_applications = {
    "stuart-chatbot" = {
      description = "Service account for stuart-chatbot API access to generative AI"
      project     = "default"
      expires_at  = "2027-05-05T00:00:00Z"
    }
    "discovery-tool" = {
      description = "Service account for discovery-tool API access to generative AI"
      project     = "default"
      expires_at  = "2027-05-05T00:00:00Z"
    }
  }
}

data "scaleway_account_project" "inference" {
  for_each = local.inference_projects
  name     = each.value
}

resource "scaleway_iam_application" "inference" {
  for_each    = local.inference_applications
  name        = "${each.key}-inference"
  description = each.value.description
}

resource "scaleway_iam_policy" "inference" {
  for_each       = local.inference_applications
  name           = "${each.key}-inference-policy"
  description    = "Grants inference access to the ${each.key} service account"
  application_id = scaleway_iam_application.inference[each.key].id

  rule {
    project_ids          = [data.scaleway_account_project.inference[each.value.project].id]
    permission_set_names = ["GenerativeApisModelAccess","GenerativeApisFullAccess"]
  }
}

resource "scaleway_iam_api_key" "inference" {
  for_each           = local.inference_applications
  application_id     = scaleway_iam_application.inference[each.key].id
  description        = "API key for ${each.key} inference access"
  expires_at         = each.value.expires_at
  default_project_id = data.scaleway_account_project.inference[each.value.project].id
}

output "inference_api_keys" {
  value     = { for k, v in scaleway_iam_api_key.inference : k => v.secret_key }
  sensitive = true
}
