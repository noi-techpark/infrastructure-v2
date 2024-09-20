<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

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

In the [eks.tf](../infrastructure/terraform/compute/eks.tf) users and service accounts can be managed through the `aws_auth_users` map. Once changes have been made they can be applied through `terraform apply`.

## Tips

A list of commonly used commands for Kubernetes can be found in the [official cheatsheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/).

## Port Forwarding 

When testing against the Kubernetes Cluster in the Cloud environment we can open an tunnel between the local machine and the workload running in the cluster with the following `port-forward` command:

```sh
kubectl port-forward <pod-name> <localport>:<remote-port>
```

EG: to forward the `Storage Mosquitto` to the `1884` port type
```sh
kubectl port-forward mosquitto-storage-c79967d5d-kcjcb 1884:1883
```

By doing so our `localhost:1884` forwards to the `pod's 1883 remote` port. 

## Authenticating with AWS SSO (Consip)
### Setup role based authentication in EKS
To allow SSO users access to EKS, the corresponding role has to first be registered in `infrastructure/terraform/kubernetes/kubernets.tf`  under `locals.aws_auth_roles`

[Follow this guide](https://repost.aws/knowledge-center/eks-configure-sso-user)
to get the role ARN and add it to the terraform script

### How to login

Log in via consip AWS SSO, when you are presented with the "AWS Account" selection screen, select "Command line of programmatic access"

following the instructions provided there to set up SSO:
```sh
aws configure sso
```
Rename the newly created AWS profile "aqcloud-98172389712" to "consip" in my aws config file
```sh
$EDITOR ~/.aws/config
```

make sure you're logged into aws
```sh
aws sso login --profile consip
```

Now create the kubectl credentials following the infrastructure-v2 documentation, but using the newly created AWS CLI profile:
```sh
aws eks --region eu-west-1 update-kubeconfig --name aws-main-eu-01 --profile consip
```

Now you should be able to issue any kubectl command
```sh
kubectl get pods -A
```


Once installed, every time kubectl complains about you not being logged in, just renew your login using
```sh
aws sso login --profile consip
```

## Use kubectx to manage multiple clusters (optional)
Kubectl lets you define contexts that you can switch between, but the UX is somewhat cumbersome.

`Kubectx` is a thin wrapper that lets you manage and switch between contexts in an intuitive way

Install [kubectx](https://github.com/ahmetb/kubectx)

### Usage
```sh
# Get the name of the contexts on your system (name defaults to the cluster ARN):
> kubectx

# Now rename the context to something more handy
> kubectx consip=<insert the ARN>

# switch to new context
> kubectx consip

# All kubectl commands you issue are always using the context you set with kubectx
```
