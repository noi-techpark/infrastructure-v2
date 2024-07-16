resource "aws_eip" "eks-ingress-b" {    
    # This is our main IP address, probably should make sure it's not given up
    lifecycle {
      prevent_destroy = true
    }
}