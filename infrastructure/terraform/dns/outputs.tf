output "main-nameservers" {
  value = aws_route53_zone.main.name_servers
  description = "List of nameserver of the main hosted zone. Add this as a NS record to wherever the base domain is hosted"
}
