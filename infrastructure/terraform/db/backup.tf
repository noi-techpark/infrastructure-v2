resource "aws_backup_plan" "rds" {
  name = "rds"
  rule {
    rule_name = "weekly"
    target_vault_name = aws_backup_vault.rds.name
    schedule = "cron(0 1 ? * 2 *)"
    start_window = 240
    completion_window = 1440

    lifecycle {
      delete_after = 365
      cold_storage_after = 10 
    }
  }
}

resource "aws_backup_vault" "rds"{
  name = "rds"
}

resource "aws_backup_selection" "rds" {
  iam_role_arn = aws_iam_role.rds_backup.arn
  name = "rds"
  plan_id = aws_backup_plan.rds.id
  
  resources = [ 
    aws_db_instance.pg-timeseries.arn
  ]
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["backup.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}
resource "aws_iam_role" "rds_backup" {
  name               = "AWSBackupDefaultServiceRole"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role_policy_attachment" "rds_backup" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup"
  role       = aws_iam_role.rds_backup.name
}