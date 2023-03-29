resource "aws_ecr_repository" "notifier" {
  name = "notifier"
}

resource "aws_ecr_repository" "kamel" {
  name = "kamel"
}

resource "aws_ecr_repository" "camel" {
  name = "camel"
}
