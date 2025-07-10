# Setup irsa to allow k8s cert-manager to change route53 for dns challenge
data "aws_eks_cluster" "default" {
  name = var.cluster_name
}

data "aws_iam_openid_connect_provider" "oidc" {
  url = data.aws_eks_cluster.default.identity[0].oidc[0].issuer
}

resource "aws_iam_policy" "cert_manager_dns" {
  name   = "cert-manager-dns01"
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect = "Allow",
      Action = [
        "route53:ChangeResourceRecordSets",
        "route53:ListResourceRecordSets",
        "route53:ListHostedZonesByName",
        "route53:GetChange",
        "route53:ListHostedZones"
      ],
      Resource = "*"
    }]
  })
}

resource "aws_iam_role" "cert_manager" {
  name = "cert-manager-dns01-irsa"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [{
      Effect = "Allow",
      Principal = {
        Federated = data.aws_iam_openid_connect_provider.oidc.arn
      },
      Action = "sts:AssumeRoleWithWebIdentity",
      Condition = {
        StringEquals = {
          "${data.aws_iam_openid_connect_provider.oidc.url}:sub" = "system:serviceaccount:cert-manager:cert-manager"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "cert_manager_attach" {
  role       = aws_iam_role.cert_manager.name
  policy_arn = aws_iam_policy.cert_manager_dns.arn
}
