resource "aws_route53_zone" "main" {
    name = var.main-zone
    force_destroy = true
}

# resource "aws_route53_record" "main-ns" {
#     zone_id = aws_route53_zone.main.zone_id
#     name = aws_route53_zone.main.name
#     type = "NS"
#     ttl = "30"
#     records = aws_route53_zone.main.name_servers
# }

resource "aws_route53_record" "ninja" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "mobility.api"
  type    = "A"
  ttl     = "7200"
  records = ["115.146.85.100"]
