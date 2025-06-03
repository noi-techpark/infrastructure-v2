variable "bucket_name" {
  description = "The name of the S3 bucket"
  type        = string
}

variable "domain" {
  description = "The domain used to tag the S3 bucket"
  type        = string
  default     = "Open Data Hub"
}

variable "index_document" {
  description = "The index document for the static website"
  type        = string
  default     = "index.html"
}

variable "error_document" {
  description = "The error document for the static website"
  type        = string
  default     = "error.html"
}

resource "aws_s3_bucket" "static_website" {
  bucket = var.bucket_name

  tags = {
    Name        = var.bucket_name
    Domain      = var.domain
  }
}

resource "aws_s3_bucket_website_configuration" "static_website" {
  bucket = aws_s3_bucket.static_website.id

  index_document {
    suffix = var.index_document
  }

  error_document {
    key = var.error_document
  }
}

resource "aws_s3_bucket_policy" "static_website_policy" {
  depends_on = [
    aws_s3_bucket_acl.static_website_acl,
  ]

  bucket = aws_s3_bucket.static_website.id

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect    = "Allow",
        Principal = "*",
        Action    = "s3:GetObject",
        Resource  = "${aws_s3_bucket.static_website.arn}/*"
      }
    ]
  })
}

////// PUBLIC ACL

resource "aws_s3_bucket_ownership_controls" "owner" {
  bucket = aws_s3_bucket.static_website.id
  rule {
    object_ownership = "BucketOwnerPreferred"
  }
}

resource "aws_s3_bucket_public_access_block" "public_acl" {
  bucket = aws_s3_bucket.static_website.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_acl" "static_website_acl" {
  depends_on = [
    aws_s3_bucket_ownership_controls.owner,
    aws_s3_bucket_public_access_block.public_acl,
  ]

  bucket = aws_s3_bucket.static_website.id
  acl    = "public-read"
}

output "bucket_name" {
  value = aws_s3_bucket.static_website.bucket
}

output "website_endpoint" {
  value = aws_s3_bucket_website_configuration.static_website.website_endpoint
}