resource "aws_ecr_repository" "notifier" {
  name = "notifier"
}

resource "aws_ecr_repository" "camel" {
  name = "camel"
}
