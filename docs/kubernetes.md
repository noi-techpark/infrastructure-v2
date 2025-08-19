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


## AWS Console

The Kubernetes cluster can be accessed from withing the [AWS Console](https://eu-west-1.console.aws.amazon.com/eks/home?region=eu-west-1#/clusters/aws-main-eu-01).

## Manage Kubernetes Users & Service Accounts

Users in EKS are not automatically managed by AWS IAM, it is therefore necessary to manually grant or revoke access to users and service accounts.

In the [env.tf](../infrastructure/terraform/compute/env.tf) users and service accounts can be managed through the `eks_admins` map. Once changes have been made they can be applied through `terraform apply`.

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

## Upgrading the cluster
Upgrading to a new kubernetes version is a frequent operation.
In the [terraform/kubernetes](../infrastructure/terraform/kubernetes/) folder:
- Update version number in [eks.tf](../infrastructure/terraform/kubernetes/eks.tf)
- `terraform apply`
- `terraform apply` again to see if any addons need further updating. repeat this until no changes are proposed
- (optional) upgrade the cluster autoscaler to the matching k8s version. Usually this is not necessary

### autoscaler
```sh
# Make sure to correctly set your cluster ARN / account ID and the image.tag version.
# Refer to https://github.com/kubernetes/autoscaler/releases/ for the latest releases
helm upgrade --install aws-cluster-autoscaler autoscaler/cluster-autoscaler \
  --values infrastructure/helm/aws-cluster-autoscaler/values.yaml \
  --set rbac.serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::828408288281:role/aws-main-eu-01-cluster-autoscaler" \
  --namespace kube-system --set image.tag="v1.33.0"
```