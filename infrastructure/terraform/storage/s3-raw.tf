################################################################################
## S3 buckets with IAM users and credentials.
##
## To add a new bucket, add an entry to local.s3_buckets:
##   "my-key" = {
##     bucket_prefix   = "opendatahub-my-key-"
##     iam_user        = "opendatahub-my-key-storage-user"
##     iam_policy_name = "OpenDataHubMyKeyStoragePolicy"
##   }
################################################################################

locals {
  s3_bucket_policy_actions = {
    read_only  = ["s3:GetObject"]
    read_write = ["s3:GetObject", "s3:PutObject"]
    full       = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
  }

  s3_buckets = {
    "opendatahub-raw" = {
      bucket_prefix   = "opendatahub-raw-"
      iam_user        = "opendatahub-s3-raw"
      iam_policy_name = "OpenDataHubS3RawPolicy"
      policy_type     = "read_write"
    }
  }
}

resource "aws_s3_bucket" "buckets" {
  for_each      = local.s3_buckets
  bucket_prefix = each.value.bucket_prefix
}

resource "aws_s3_bucket_public_access_block" "buckets" {
  for_each = local.s3_buckets
  bucket   = aws_s3_bucket.buckets[each.key].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "buckets" {
  for_each = local.s3_buckets
  bucket   = aws_s3_bucket.buckets[each.key].id

  versioning_configuration {
    status = "Enabled"
  }
}

data "aws_iam_policy_document" "bucket_access" {
  for_each = local.s3_buckets

  statement {
    sid       = "listbucket"
    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.buckets[each.key].arn]
  }

  statement {
    sid = "readwrite"
    actions = local.s3_bucket_policy_actions[each.value.policy_type]
    resources = ["${aws_s3_bucket.buckets[each.key].arn}/*"]
  }
}


resource "aws_iam_policy" "bucket_access" {
  for_each = local.s3_buckets
  name     = each.value.iam_policy_name
  policy   = data.aws_iam_policy_document.bucket_access[each.key].json
}

resource "aws_iam_user" "bucket_access" {
  for_each = local.s3_buckets
  name     = each.value.iam_user
}

resource "aws_iam_user_policy_attachment" "bucket_access" {
  for_each   = local.s3_buckets
  user       = aws_iam_user.bucket_access[each.key].name
  policy_arn = aws_iam_policy.bucket_access[each.key].arn
}

resource "aws_iam_access_key" "bucket_access" {
  for_each = local.s3_buckets
  user     = aws_iam_user.bucket_access[each.key].name
}
