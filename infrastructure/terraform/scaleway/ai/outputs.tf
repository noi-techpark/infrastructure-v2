output "chatbot_inference_endpoints" {
  description = "Public HTTPS endpoints of the chatbot inference deployments"
  value       = [for d in scaleway_inference_deployment.chatbot : d.public_endpoint[0].url]
}

output "chatbot_api_key_access_key" {
  description = "Access key ID for the chatbot IAM application"
  value       = scaleway_iam_api_key.chatbot.access_key
}

output "chatbot_api_key_secret_key" {
  description = "Secret key for the chatbot IAM application (use with the inference API)"
  value       = scaleway_iam_api_key.chatbot.secret_key
  sensitive   = true
}
