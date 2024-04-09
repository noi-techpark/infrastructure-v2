# This file creates users for services

# User for github actions ci/cd access
resource "aws_iam_user" "github-actions" {
  name = "github-actions"
}