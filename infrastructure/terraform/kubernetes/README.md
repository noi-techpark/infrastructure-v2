<!--
SPDX-FileCopyrightText: 2023 NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Terraform Cloud Workspace

This workspace contains all resources associated with computing infrastructure (ECR, EKS, VPC, and so on).

## Setup

On the very first run that creates the cluster, you must set the variable 

`terraform apply -var "INITIAL_CREATE=true"`

and then run apply a second time without the variable set.  
This is because some resources depend on the cluster already existing (e.g. kubernetes_manifest) and will error otherwise.

### AWS Credentials

The Terraform Cloud Workspace must define the following two environment variables:

- `AWS_ACCESS_KEY_ID` (sensitive)
- `AWS_SECRET_ACCESS_KEY` (sensitive)

Optionally, you can set an
- `ENVIRONMENT` distinguishes between production and dev environments, mainly regarding provisioning size and user accounts
- `INITIAL_CREATE` skip resources that depend on cluster already existing
