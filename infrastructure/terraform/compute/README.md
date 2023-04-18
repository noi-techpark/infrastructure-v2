# Terraform Cloud Workspace

This workspace contains all resources associated with computing infrastructure (ECR, EKS, VPC, and so on).

## Setup

### AWS Credentials

The Terraform Cloud Workspace must define the following two environment variables:

- `AWS_ACCESS_KEY_ID` (sensitive)
- `AWS_SECRET_ACCESS_KEY` (sensitive)

### Remote State Sharing

The workspace must be configured to share outputs with the whole organization:

```
Terraform Cloud UI > Workspace > Settings > General > Remote state sharing > Share with all workspaces in this organization
```

The configuration can be found by visiting one of the following direct links:

```
https://app.terraform.io/app/noi-digital/workspaces/opendatahub-compute-development/settings/general (development)
```

```
https://app.terraform.io/app/noi-digital/workspaces/opendatahub-compute-production/settings/general (production)
```

This step is necessary to allow the `opendatahub-kubernetes-*` to consume the outputs of this workspace and deploy additional resources.

See [kubernetes / backend.tf](../kubernetes/backend.tf) for additional information:

```hcl
data "tfe_outputs" "compute" {
  organization = "noi-digital"
  workspace    = "opendatahub-compute-development"
}
```

Official documentation on remote states:

- https://developer.hashicorp.com/terraform/language/state/remote-state-data
- https://registry.terraform.io/providers/hashicorp/tfe/latest/docs/data-sources/outputs
