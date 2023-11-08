<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Helm

Helm is used within the Open-Data Hub orchestration management as package manager for Kubernetes resources.

## Prerequisites

In order to be able to plan, apply, and debug infrastructure provisioning, it is necessary to satisfy the following prerequisites:

- Installation of [Helm CLI](https://helm.sh/docs/intro/install/)
- Prerequisites of [Kubernetes](kubernetes.md#Prerequisites)

## Namespaces

create the namespaces the helm charts are deployed to:
```
kubectl create namespace core
kubectl create namespace collectors
```
## Packages

### Kubernetes Dashboard

https://artifacthub.io/packages/helm/metrics-server/metrics-server

```sh
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
```

```sh
helm upgrade --install \
  metrics-server metrics-server/metrics-server \
  --values infrastructure/helm/metrics-server/values.yaml \
  --namespace kube-system
```

https://artifacthub.io/packages/helm/k8s-dashboard/kubernetes-dashboard

```sh
helm repo add kubernetes-dashboard https://kubernetes.github.io/dashboard/
```

```sh
helm upgrade --install \
  kubernetes-dashboard kubernetes-dashboard/kubernetes-dashboard \
  --values infrastructure/helm/kubernetes-dashboard/values.yaml \
  --namespace kube-system
```

### Amazon ALB in EKS

https://docs.aws.amazon.com/eks/latest/userguide/aws-load-balancer-controller.html

```sh
helm repo add eks https://aws.github.io/eks-charts
```

```sh
helm upgrade --install aws-load-balancer-controller eks/aws-load-balancer-controller \
  --values infrastructure/helm/aws-load-balancer-controller/values.yaml \
  --namespace kube-system
```

### MongoDB

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
```

```sh
helm upgrade --install mongodb bitnami/mongodb \
  --values infrastructure/helm/mongodb/values.yaml \
  --namespace core
```

### RabbitMQ

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
```

```sh
helm upgrade --install rabbitmq bitnami/rabbitmq \
  --values infrastructure/helm/rabbitmq/values.yaml \
  --namespace core
```

### Camel K

**Reminder: update the following credentials `[USER]` and `[TOKEN]` with valid docker registry credentials.**

**Tips: install `Kamel CLI` and read the Camel K [documentation](https://camel.apache.org/camel-k/1.9.x/running/running.html).**

```sh
kubectl create secret docker-registry docker-secrets \
  --docker-username=[USER] \
  --docker-password=[TOKEN]\
  --namespace core

# !!! Amazon ECR is not supported yet by Kamel, use Docker Hub instead.
# !!! See https://github.com/apache/camel-k/issues/4107.

# kubectl create secret docker-registry docker-secrets \
#   --docker-server=463112166163.dkr.ecr.eu-west-1.amazonaws.com \
#   --docker-username=AWS \
#   --docker-password=$(aws ecr get-login-password --region eu-west-1)
```

```sh
helm repo add camel-k https://apache.github.io/camel-k/charts
```

```sh

helm upgrade --install camel-k camel-k/camel-k \
  --values infrastructure/helm/camel-k/values.yaml
  --version 0.13.1 \
  --namespace core
```


### Mosquitto

```sh
helm repo add naps https://naps.github.io/helm-charts/
```

```sh
helm upgrade --install mosquitto naps/mosquitto \
  --values infrastructure/helm/mosquitto/values.yaml \
  --namespace core
```

### Notifier

**Reminder: authenticate to docker via `aws ecr get-login-password` and `docker login` before invoking the `docker push` command.**

See [https://docs.aws.amazon.com/AmazonECR/latest/userguide/getting-started-cli.html](Using Amazon ECR with the AWS CLI).

**Reminder: update the `image.repository` configuration value.**

**Reminder: update the `RABBITMQ_CLUSTER_URL` environment variable with valid RabbitMQ credentials.**

```sh
aws ecr get-login-password --region [REGION] | docker login --username AWS --password-stdin [ACCOUNT].dkr.ecr.[REGION].amazonaws.com
# See https://docs.aws.amazon.com/AmazonECR/latest/userguide/getting-started-cli.html for more information.

docker build -t [ACCOUNT].dkr.ecr.[REGION].amazonaws.com/notifier:latest .
docker push [ACCOUNT].dkr.ecr.[REGION].amazonaws.com/notifier:latest
```

```sh
helm upgrade --install notifier ./infrastructure/helm/notifier/notifier \
  --values infrastructure/helm/notifier/values.yaml \
  --namespace core
```

### Nginx ingress
TODO: values.yaml contains hardcoded subnet of dev environment.
```sh
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx  --namespace ingress-nginx --create-namespace \
--values infrastructure/helm/nginx-ingress/values.yaml \
--set "controller.service.annotations.service\.beta\.kubernetes\.io/aws-load-balancer-eip-allocations=eipalloc-0b84603c6d3f425bf"
```

### Certmanager (https certificates)

```sh
helm repo add jetstack https://charts.jetstack.io
helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.12.4 \
  --set installCRDs=true
  
# Create the letsencrypt issuers. 
# TODO: create a route53 issuer so we can use dns instead of http challenges
kubectl create -f infrastructure/ingress/cert-manager/letsencrypt-staging-clusterissuer.yaml
kubectl create -f infrastructure/ingress/cert-manager/letsencrypt-prod-clusterissuer.yaml
```

## Open Data Hub core applications
These are the legacy core applications of the Open Data Hub
These use already existing containers and pipelines and are configured via env variables set as "env." values.
Image tags are hardcoded, so upgrading the applications has to be done by updating image tags (commit hash)
### Ninja API
Outbound mobility API used to query mobility data
TODO: rewrite to /v2/ context and add the STA mobility proxies
```sh
helm upgrade --install ninja-api infrastructure/helm/ninja-api/ninja-api --namespace core --values infrastructure/helm/ninja-api/values.yaml --set env.DB_PASSWORD=********
```
### Mobility core / writer
Endpoint where data collectors and elaborations push their mobility data. Also maintains and versions the mobility database schema
```sh
helm upgrade --install bdp-core infrastructure/helm/bdp-core/bdp-core --namespace core --values infrastructure/helm/bdp-core/values.yaml --set env.DB_PASSWORD=********
```
### Analytics
Frontend application that uses Ninja-API to visualize mobility data on maps and charts
```sh
helm upgrade --install analytics infrastructure/helm/analytics/analytics --namespace core --values infrastructure/helm/analytics/values.yaml
```