################################################################################
## This file manages the building server instance.
## Instance type: PLAY2-NANO (2 vCPU, 4 GB RAM) + 50 GB SBS data volume
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/instance_server
################################################################################

data "scaleway_account_project" "opendatahub_apps" {
  name = "Open Data Hub - Apps"
}

data "scaleway_marketplace_image" "debian" {
  label         = "debian_trixie"
  zone          = "fr-par-1"
}

resource "scaleway_instance_ip" "building" {
  project_id = data.scaleway_account_project.opendatahub_apps.id
  zone       = "fr-par-1"
}

resource "scaleway_instance_security_group" "building" {
  project_id              = data.scaleway_account_project.opendatahub_apps.id
  zone                    = "fr-par-1"
  name                    = "building"
  description             = "Security group for building server"
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
    port     = "443"
    protocol = "TCP"
  }

  inbound_rule {
    action   = "accept"
    protocol = "ICMP"
  }
}


resource "scaleway_instance_server" "building" {
  project_id            = data.scaleway_account_project.opendatahub_apps.id
  zone                  = "fr-par-1"
  name                  = "building"
  type                  = "DEV1-M"
  image                 = data.scaleway_marketplace_image.debian.id
  ip_id                 = scaleway_instance_ip.building.id
  security_group_id     = scaleway_instance_security_group.building.id

  root_volume {
    delete_on_termination = true
    size_in_gb = 20
  }

  additional_volume_ids = [scaleway_block_volume.building_data.id]

  tags = ["building"]
}

resource "scaleway_block_volume" "building_data" {
  size_in_gb = 50
  iops       = 5000
}

output "building_public_ip" {
  value       = scaleway_instance_ip.building.address
  description = "Public IP address of the building server"
}
