# Helm

Helm is used within the Open-Data Hub orchestration management as package manager for Kubernetes resources.

## Prerequisites

In order to be able to plan, apply, and debug infrastructure provisioning, it is necessary to satisfy the following prerequisites:

- Installation of [Helm CLI](https://helm.sh/docs/intro/install/)
- Prerequisites of [Kubernetes](kubernetes.md#Prerequisites)

## Packages

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

**Reminder: update the following credentials `[USER]` and `[TOKEN]` with valid docker registry credentials.**

**Tips: install `Kamel CLI` and read the Camel K [documentation](https://camel.apache.org/camel-k/1.9.x/running/running.html).**

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

**Reminder: update the following repository `[REPOSITORY]` with a valid docker registry repository.**

**Reminder: authenticate to docker via `docker login` before invoking the `docker push` command.**

```
docker build -t [REPOSITORY]/notifier:latest .
docker push [REPOSITORY]/notifier:latest
```

```
helm upgrade --install notifier ./infrastructure/helm/notifier/notifier \
  --values infrastructure/helm/notifier/values.yaml
```
