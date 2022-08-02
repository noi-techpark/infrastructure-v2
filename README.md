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

TODO: move docker repository to ECR once we have permissions.

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
cd infrastructure
```

```
kamel run \
  --name mqtt-route \
  --property mqtt.url=tcp://mosquitto-edge:1883 \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    inbound/src/main/java/it/bz/opendatahub/inbound/MqttRoute.java
```

```
kamel run \
  --name rest-route \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    inbound/src/main/java/it/bz/opendatahub/inbound/RestRoute.java
```

```
kamel run \
  --name writer-route \
  --property quarkus.mongodb.connection-string=mongodb://mongodb-0.mongodb-headless.default.svc.cluster.local:27017 \
  --property quarkus.mongodb.devservices.enabled=false \
  --property internal_mqtt.url=tcp://mosquitto-storage.default.svc.cluster.local:1883 \
  --property internal_mqtt.topic=storage \
    writer/src/main/java/it/bz/opendatahub/writer/WriterRoute.java
```

### Mosquitto

```
helm repo add k8s-at-home https://k8s-at-home.com/charts/
```

```
helm upgrade --install mosquitto-edge k8s-at-home/mosquitto \
  --values infrastructure/helm/mosquitto/values-edge.yaml

helm upgrade --install mosquitto-notifier k8s-at-home/mosquitto \
  --values infrastructure/helm/mosquitto/values-notifier.yaml

helm upgrade --install mosquitto-storage k8s-at-home/mosquitto \
  --values infrastructure/helm/mosquitto/values-storage.yaml
```

### Notifier

TODO: move docker repository to ECR once we have permissions.

```
docker build -t roggia/notifier:latest .
docker push roggia/notifier:latest
```

```
helm upgrade --install notifier ./infrastructure/helm/notifier/notifier \
  --values infrastructure/helm/notifier/values.yaml
```
