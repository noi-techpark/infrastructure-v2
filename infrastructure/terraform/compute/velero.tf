data "aws_iam_policy_document" "velero_oidc_role_document" {
  statement {
    sid = "serviceaccount"

    actions = [
      "sts:AssumeRoleWithWebIdentity",
    ]

    principals {
      type        = "Federated"
      identifiers = ["arn:aws:iam::${local.account_id}:oidc-provider/${local.oidc_provider}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider}:sub"

      values = [
        "system:serviceaccount:velero-system:velero-server",
      ]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider}:aud"

      values = [
        "sts.amazonaws.com",
      ]
    }
  }
}

data "aws_iam_policy_document" "velero_policy_document" {
  statement {
    sid = "ec2"

    actions = [
      "ec2:DescribeVolumes",
      "ec2:DescribeSnapshots",
      "ec2:CreateTags",
      "ec2:CreateVolume",
      "ec2:CreateSnapshot",
      "ec2:DeleteSnapshot",
    ]

    resources = ["*", ]
  }
  statement {
    sid = "s3list"

    actions = [
      "s3:ListBucket",
    ]

    resources = [aws_s3_bucket.velero.arn]
  }

  statement {
    sid = "s3backup"

    actions = [
      "s3:GetObject",
      "s3:DeleteObject",
      "s3:PutObject",
      "s3:AbortMultipartUpload",
      "s3:ListMultipartUploadParts"
    ]
    resources = ["${aws_s3_bucket.velero.arn}/*", ]
  }
}

resource "aws_iam_policy" "velero_policy" {
  name   = "AmazonEKSVeleroIAMPolicy"
  policy = data.aws_iam_policy_document.velero_policy_document.json
}

resource "aws_iam_role" "velero_role" {
  name = "AmazonEKSVeleroRole"
  assume_role_policy = data.aws_iam_policy_document.velero_oidc_role_document.json
}

resource "aws_iam_role_policy_attachment" "velero_attach_iam_policy" {
  role       = aws_iam_role.velero_role.name
  policy_arn = aws_iam_policy.velero_policy.arn
}

resource "aws_s3_bucket" "velero" {
  bucket_prefix = "velero-opendatahub-"
  
  # prevent that one bucket in testing from being deleted on every apply.
  # once the bucket velero-opendatahub-bfe86313 isn't used anymore, delete this
  lifecycle {
    ignore_changes = [bucket_prefix]
  }
}
