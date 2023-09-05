output "main-nameservers" {
  value = aws_route53_zone.main.name_servers
  description = "List of nameserver of the main hosted zone. Add this as a NS record to wherever the base domain is hosted"
}

output "eks-ingress-ip" {
  value = [aws_eip.eks-ingress-a.public_ip, aws_eip.eks-ingress-b.public_ip, aws_eip.eks-ingress-c.public_ip]
  description = "Static EIB IP address for eks ingress load balancer. This is where the DNS entries point to"
}
output "eks-ingress-name" {
  value = [aws_eip.eks-ingress-a.allocation_id, aws_eip.eks-ingress-b.allocation_id, aws_eip.eks-ingress-c.allocation_id]
  description = "Static EIB IP address for eks ingress load balancer. This is where the DNS entries point to"
}