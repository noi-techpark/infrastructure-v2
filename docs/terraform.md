<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Terraform

The Open-Data Hub infrastructure is provisioned through Terraform and its state is stored in [Terraform Cloud](https://app.terraform.io/app/noi-digital/workspaces/opendatahub-v2).

## Prerequisites

In order to be able to plan, apply, and debug infrastructure provisioning, it is necessary to satisfy the following prerequisites:

- Installation of [Terraform CLI](https://learn.hashicorp.com/tutorials/terraform/install-cli)
- Authenticate to Terraform Cloud via `terraform login`

## Planning and Applying

Navigate to the Terraform folder and initialize the workspace once, then safely plan apply changes. 

```
cd infrastructure/terraform

terraform init    # Initialize the workspace - needs to be invoked only once.
terraform plan    # Prepare a new plan and review changes (read-only).
terraform apply   # Review the changes one last time and confirm before applying.
```

## Documentation

All files include a header with a link to the documentation page of the resources being deployed to the cloud.

## First setup