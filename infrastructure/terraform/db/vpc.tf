# Reference compute/vpc.tf for the subnets targeted here

data "aws_vpc" "k8s-vpc" {
  filter {
    name   = "tag:Name"
    values = ["internal-vpc"]
  }
}

data "aws_subnets" "public" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.k8s-vpc.id]
  }
  filter {
    name   = "tag:Name"
    values = ["internal-vpc-public-*"]
  }
}
data "aws_subnets" "private" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.k8s-vpc.id]
  }
  filter {
    name   = "tag:Name"
    values = ["internal-vpc-private-*"]
  }
}
