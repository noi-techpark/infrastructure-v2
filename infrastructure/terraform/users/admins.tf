# This file creates group and users for the noi digital team

resource "aws_iam_group" "admins" {
  name = "Admins"
}

resource "aws_iam_group_policy_attachment" "admin_policy_attach" {
  group = aws_iam_group.admins.name
  for_each = toset([ 
    "arn:aws:iam::aws:policy/AdministratorAccess", 
    "arn:aws:iam::aws:policy/IAMUserChangePassword"])
  policy_arn = each.value
}

locals {
  prod = var.ENVIRONMENT == "prod"
  admins_dev = [
    "c.zagler",
    "l.miotto",
    "m.rabanser",
    "p.ohnewein",
    "r.cavaliere",
    "r.thoeni",
    "s.seppi",
    "l.pasquarelli",
  ]
  admins_prod = [
    "c.zagler",
    "l.miotto",
    "p.ohnewein",
    "r.cavaliere",
    "r.thoeni",
    "s.seppi",
    "l.pasquarelli",
  ]
  admins = local.prod ? local.admins_prod : local.admins_dev
}

resource "aws_iam_user" "admins" {
  for_each = toset(local.admins)
  name = each.value
}

resource "aws_iam_group_membership" "admin_members" {
  name = "admin_membership"
  users = values(aws_iam_user.admins)[*].name
  group = aws_iam_group.admins.name
}