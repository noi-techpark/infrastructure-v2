################################################################################
## This file manages Scaleway Managed Inference deployments and the IAM
## application used to access them via API.
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/inference_deployment
################################################################################

resource "scaleway_inference_deployment" "chatbot" {
  count      = var.INFERENCE_DEPLOYMENT_COUNT
  name       = var.INFERENCE_DEPLOYMENT_COUNT == 1 ? "chatbot" : "chatbot-${count.index}"
  node_type  = var.INFERENCE_NODE_TYPE
  model_name = var.INFERENCE_MODEL_NAME
  accept_eula = true

  public_endpoint {
    is_enabled = true
  }
}

# IAM application used as a service account to call the inference API.
resource "scaleway_iam_application" "chatbot" {
  name        = "chatbot-inference"
  description = "Service account for chatbot API access to managed inference"
}

resource "scaleway_iam_policy" "chatbot_inference" {
  name           = "chatbot-inference-policy"
  description    = "Grants inference access to the chatbot service account"
  application_id = scaleway_iam_application.chatbot.id

  rule {
    project_ids          = [var.SCW_PROJECT_ID]
    permission_set_names = ["InferenceFullAccess"]
  }
}

resource "scaleway_iam_api_key" "chatbot" {
  application_id = scaleway_iam_application.chatbot.id
  description    = "API key for chatbot inference access"
}
