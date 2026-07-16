################################################################################
## This file manages the SSH key pair for the digital sovereignty development server.
## NOTE: The private key is stored in Terraform state — keep the backend secure.
## https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key
################################################################################

resource "tls_private_key" "sovereign" {
  algorithm = "ED25519"
}

resource "scaleway_account_ssh_key" "sovereign" {
  name       = "sovereign"
  public_key = tls_private_key.sovereign.public_key_openssh
  project_id = scaleway_account_project.it.id
}

output "sovereign_ssh_private_key" {
  value       = tls_private_key.sovereign.private_key_openssh
  sensitive   = true
  description = "Private SSH key for the digital sovereignty development server (ED25519)"
}
