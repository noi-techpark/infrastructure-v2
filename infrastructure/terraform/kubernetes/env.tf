# Environment specific settings, currently only distinguishing between dev and production
locals {
  prod = var.ENVIRONMENT == "prod"

  eks_main_pool_instance_types = local.prod ? ["t3a.large"] : ["r7a.large"]

  eks_admins_dev = [
    "c.zagler",
    "m.rabanser",
    "r.thoeni",
    "s.seppi",
    "l.pasquarelli",
    "m.roggia",
  ]
  eks_admins_prod = [
    "c.zagler",
    "r.thoeni",
    "s.seppi",
    "l.pasquarelli",
    "m.roggia",
  ]
  eks_admins = local.prod ? local.eks_admins_prod : local.eks_admins_dev
}