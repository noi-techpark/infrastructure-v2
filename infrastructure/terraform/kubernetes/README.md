<!--
SPDX-FileCopyrightText: 2023 NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Terraform Cloud Workspace

This workspace contains all resources associated with kubernetes infrastructure (addons, authentication, load balancing, and so on).

## Setup

### AWS Credentials

The Terraform Cloud Workspace must define the following two environment variables:

- `AWS_ACCESS_KEY_ID` (sensitive)
- `AWS_SECRET_ACCESS_KEY` (sensitive)
