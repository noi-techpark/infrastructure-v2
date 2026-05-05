################################################################################
## This file manages the SSH key pair for the building server.
## NOTE: The private key is stored in Terraform state — keep the backend secure.
## https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key
################################################################################

resource "tls_private_key" "building" {
  algorithm = "ED25519"
}

resource "scaleway_account_ssh_key" "building" {
  name       = "building"
  public_key = tls_private_key.building.public_key_openssh
  project_id = data.scaleway_account_project.opendatahub_apps.id
}

output "building_ssh_private_key" {
  value       = tls_private_key.building.private_key_openssh
  sensitive   = true
  description = "Private SSH key for the building server (ED25519)"
}
