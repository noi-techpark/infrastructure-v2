# Open-Data Hub Infrastructure

> This repository contains the new infrastructure architecture as described in the document prepared by Animeshon and approved by the owners of the project.

## Prerequisites

- AWS CLI
- Helm CLI
- Terraform CLI
- Kubernetes CLI

## Links

- [Terraform Cloud](https://app.terraform.io/app/noi-digital/workspaces/opendatahub-v2)
- [Architecture]()
- [Kubernetes Dashboard](http://k8s-default-kubernet-62841e8dd0-301015478.eu-west-1.elb.amazonaws.com/)
- [Amazon EKS](https://eu-west-1.console.aws.amazon.com/eks/home?region=eu-west-1#/clusters/aws-main-eu-01)

## Terraform

- EKS
- Amazon SNS
- Amazon IoT

## Kubernetes & Helm

- Dashboard
- DataDog
- Apache Camel K
- MongoDB
- Change Stream Service

## Monitoring & Tracing

- [Kubernetes]()
- [MongoDB]()
- [Apache Camel K]()
- [Amazon SNS]()

## Guides

- [Terraform provisioning from scratch]()
- [Setup kubectl via AWS CLI]()
- [Add new user to Kubernetes]()
- [Expose services via ELB or ALB]()

## How to

### Terraform

```
cd infrastructure/terraform

terraform init
terraform plan
```

### Kubernetes Dashboard

https://artifacthub.io/packages/helm/metrics-server/metrics-server

```
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
```

```
helm upgrade --install \
  metrics-server metrics-server/metrics-server \
  --values infrastructure/helm/metrics-server/values.yaml
```

https://artifacthub.io/packages/helm/k8s-dashboard/kubernetes-dashboard

```
helm repo add kubernetes-dashboard https://kubernetes.github.io/dashboard/
```

```
helm upgrade --install \
  kubernetes-dashboard kubernetes-dashboard/kubernetes-dashboard \
  --values infrastructure/helm/kubernetes-dashboard/values.yaml
```

### Amazon ALB in EKS

https://docs.aws.amazon.com/eks/latest/userguide/aws-load-balancer-controller.html

```
helm repo add eks https://aws.github.io/eks-charts
```

```
helm upgrade --install aws-load-balancer-controller eks/aws-load-balancer-controller \
  --values infrastructure/helm/kubernetes-dashboard/values.yaml \
  --namespace kube-system
```

### MongoDB

```
helm repo add bitnami https://charts.bitnami.com/bitnami
```

```
helm upgrade --install mongodb bitnami/mongodb \
  --values infrastructure/helm/mongodb/values.yaml
```

### Camel K

```
kubectl create secret docker-registry docker-hub-secrets \
  --docker-username=[USER] \
  --docker-password=[TOKEN]
```

```
helm repo add camel-k https://apache.github.io/camel-k/charts
```

```
helm upgrade --install camel-k camel-k/camel-k \
  --values infrastructure/helm/camel-k/values.yaml
```

Tips: install `kamel` CLI and read Camel K [docs](https://camel.apache.org/camel-k/1.9.x/running/running.html).

#### Run Camel Routes

```
cd infrastructure/apache-camel
```

```
kamel run \
  --name queue-route \
  --property mqtt.url=tcp://mosquitto:1883 \
  --property queue_internal_storage.url=tcp://mosquittostorage:1883 \
  --property queue_internal_storage.topic=storage \
    QueueRoute.java
```
