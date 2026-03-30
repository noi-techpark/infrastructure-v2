################################################################################
## S3 bucket and IRSA role for s3-raw
################################################################################

resource "aws_s3_bucket" "s3_raw" {
  bucket_prefix = "opendatahub-raw-"
}

resource "aws_s3_bucket_public_access_block" "s3_raw" {
  bucket = aws_s3_bucket.s3_raw.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "s3_raw" {
  bucket = aws_s3_bucket.s3_raw.id

  versioning_configuration {
    status = "Enabled"
  }
}

data "aws_iam_policy_document" "s3_raw_policy_document" {
  statement {
    sid     = "listbucket"
    actions = ["s3:ListBucket"]
    resources = [aws_s3_bucket.s3_raw.arn]
  }

  statement {
    sid = "readwrite"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = ["${aws_s3_bucket.s3_raw.arn}/*"]
  }
}

resource "aws_iam_policy" "s3_raw_policy" {
  name   = "S3RawPolicy"
  policy = data.aws_iam_policy_document.s3_raw_policy_document.json
}

data "aws_iam_policy_document" "s3_raw_oidc_role_document" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = ["arn:aws:iam::${local.account_id}:oidc-provider/${local.oidc_provider}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider}:sub"
      values   = ["system:serviceaccount:core:raw-writer-2"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "s3_raw_role" {
  name               = "S3RawRole"
  assume_role_policy = data.aws_iam_policy_document.s3_raw_oidc_role_document.json
}

resource "aws_iam_role_policy_attachment" "s3_raw_attach" {
  role       = aws_iam_role.s3_raw_role.name
  policy_arn = aws_iam_policy.s3_raw_policy.arn
}

output "s3_raw_role_arn" {
  value = aws_iam_role.s3_raw_role.arn
}

output "s3_raw_bucket_name" {
  value = aws_s3_bucket.s3_raw.id
}
