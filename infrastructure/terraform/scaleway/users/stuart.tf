################################################################################
## This file manages Scaleway Managed Inference deployments and the IAM
## application used to access them via API.
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/inference_deployment
################################################################################


data "scaleway_account_project" "opendatahub_apps" {
  name = "Open Data Hub - Apps"
}

# IAM application used as a service account to call the inference API.
resource "scaleway_iam_application" "stuart_chatbot" {
  name        = "stuart-chatbot-inference"
  description = "Service account for stuart-chatbot API access to generative AI"
}

resource "scaleway_iam_policy" "stuart_chatbot_inference" {
  name           = "stuart-chatbot-inference-policy"
  description    = "Grants inference access to the stuart chatbot service account"
  application_id = scaleway_iam_application.stuart_chatbot.id

  rule {
    project_ids          = [data.scaleway_account_project.opendatahub_apps.id]
    permission_set_names = ["GenerativeApisModelAccess"]
  }
}

resource "scaleway_iam_api_key" "stuart_chatbot" {
  application_id = scaleway_iam_application.stuart_chatbot.id
  description    = "API key for stuart inference access"
  expires_at     = "2027-05-05T00:00:00Z"
}

output "stuart_chatbot_api_key" {
  value     = scaleway_iam_api_key.stuart_chatbot.secret_key
  sensitive = true
}
