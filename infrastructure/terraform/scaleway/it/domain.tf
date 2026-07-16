################################################################################
## This file registers the noicloud.eu domain and points it at the digital
## sovereignty development server.
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/domain_registration
## https://registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/domain_record
################################################################################

resource "scaleway_domain_registration" "noicloud" {
  domain_names = ["noicloud.eu"]
  project_id   = scaleway_account_project.it.id
  auto_renew   = true

  # Contact changes require a domain trade (not a Terraform-managed operation);
  # this references the contact already registered via BuyDomains instead of
  # restating its fields, which the provider refuses to reconcile after creation.
  owner_contact_id = "qBq-PH2NQXWcFHtj_7F8ICRHH4p5aJLmG6FXIWMSoyvjMKM="
}

resource "scaleway_domain_record" "noicloud_a" {
  dns_zone = "noicloud.eu"
  name     = ""
  type     = "A"
  data     = scaleway_instance_ip.sovereign.address
  ttl      = 300

  depends_on = [scaleway_domain_registration.noicloud]
}
