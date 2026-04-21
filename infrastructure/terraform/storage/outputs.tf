################################################################################
## Outputs
################################################################################

output "buckets" {
  sensitive = true
  value = { for k, v in aws_s3_bucket.buckets : k => {
    bucket_name       = v.id
    access_key_id     = aws_iam_access_key.bucket_access[k].id
    secret_access_key = aws_iam_access_key.bucket_access[k].secret
  }}
}
