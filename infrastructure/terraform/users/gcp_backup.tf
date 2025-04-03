# IAM User for gcp RDS backup system
resource "aws_iam_user_policy_attachment" "gcp_backup_rds" {
  for_each = toset([ 
    "arn:aws:iam::aws:policy/AmazonRDSReadOnlyAccess"])
  policy_arn = each.value
  user = aws_iam_user.gcp_backup_rds.name
}

resource "aws_iam_user" "gcp_backup_rds" {
  name = "gcp_backup_rds"
}

resource "aws_iam_access_key" "gcp_backup_rds" {
  user = aws_iam_user.gcp_backup_rds.name
}