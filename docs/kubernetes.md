# Kubernetes

The Open-Data Hub orchestration is managed through Kubernetes and provisioned through [AWS EKS](https://eu-west-1.console.aws.amazon.com/eks/home?region=eu-west-1#/clusters/aws-main-eu-01).

## Prerequisites

In order to be able to managed Kubernetes orchestration, it is necessary to satisfy the following prerequisites:

- Installation of [Kuberentes CLI](https://kubernetes.io/docs/tasks/tools/)
- Authenticate to AWS EKS via `aws eks --region eu-west-1 update-kubeconfig --name aws-main-eu-01`


## Dashboard

On top of the dashboard provided by AWS EKS, a standard [Kuberenetes Dashboard](http://k8s-default-kubernet-62841e8dd0-731115856.eu-west-1.elb.amazonaws.com/) has been deployed with read-only access to resources within the cluster.

**Warning: This dashboard is exposed through AWS ALB without authentication.**

## AWS Console

The Kubernetes cluster can be accessed from withing the [AWS Console](https://eu-west-1.console.aws.amazon.com/eks/home?region=eu-west-1#/clusters/aws-main-eu-01).

## Manage Kubernetes Users & Service Accounts

Users in EKS are not automatically managed by AWS IAM, it is therefore necessary to manually grant or revoke access to users and service accounts.

In the [eks.tf](infrastructure/terraform/eks.tf) users and service accounts can be managed through the `aws_auth_users` map. Once changes have been made they can be applied through `terraform apply`.
