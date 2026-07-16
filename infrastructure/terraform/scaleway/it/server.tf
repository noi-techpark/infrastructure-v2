################################################################################
## This file manages the digital sovereignty development server instance.
## Instance type: DEV1-L (4 vCPU, 8 GB RAM) + 100 GB root volume
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/instance_server
################################################################################

data "scaleway_marketplace_image" "debian" {
  label = "debian_trixie"
  zone  = "fr-par-1"
}

resource "scaleway_instance_ip" "sovereign" {
  project_id = scaleway_account_project.it.id
  zone       = "fr-par-1"
}

resource "scaleway_instance_security_group" "sovereign" {
  project_id              = scaleway_account_project.it.id
  zone                    = "fr-par-1"
  name                    = "sovereign"
  description             = "Security group for the digital sovereignty development server"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"

  inbound_rule {
    action   = "accept"
    port     = "22"
    protocol = "TCP"
  }

  inbound_rule {
    action   = "accept"
    port     = "80"
    protocol = "TCP"
  }

  inbound_rule {
    action   = "accept"
    port     = "8080"
    protocol = "TCP"
  }

  inbound_rule {
    action   = "accept"
    port     = "3478"
    protocol = "TCP"
  }

  inbound_rule {
    action   = "accept"
    port     = "3478"
    protocol = "UDP"
  }

  inbound_rule {
    action   = "accept"
    port     = "443"
    protocol = "TCP"
  }

  inbound_rule {
    action   = "accept"
    protocol = "ICMP"
  }
}

resource "scaleway_instance_server" "sovereign" {
  project_id        = scaleway_account_project.it.id
  zone              = "fr-par-1"
  name              = "sovereign"
  type              = "DEV1-L"
  image             = data.scaleway_marketplace_image.debian.id
  ip_id             = scaleway_instance_ip.sovereign.id
  security_group_id = scaleway_instance_security_group.sovereign.id

  root_volume {
    delete_on_termination = true
    size_in_gb            = 80
  }

  tags = ["it", "sovereignty", "dev"]

  lifecycle {
    ignore_changes = [image]
  }
}

output "sovereign_public_ip" {
  value       = scaleway_instance_ip.sovereign.address
  description = "Public IP address of the digital sovereignty development server"
}
