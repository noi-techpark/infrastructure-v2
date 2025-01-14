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
```sh
# core services like rabbitmq, mongodb, APIs etc.
kubectl create namespace core
# data collectors and transformers
kubectl create namespace collector
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

### Amazon Cluster Autoscaler

https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws/README.md

```sh
helm repo add autoscaler https://kubernetes.github.io/autoscaler
```
```sh
# NOTE: The role ARN can be obtained from the outputs of terraform `cluster_autscaler_role`
# But it should suffice to plug in the correct account ID
helm upgrade --install aws-cluster-autoscaler autoscaler/cluster-autoscaler \
  --values infrastructure/helm/aws-cluster-autoscaler/values.yaml \
  --set rbac.serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::828408288281:role/aws-main-eu-01-cluster-autoscaler" \
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

### Velero

https://aws.amazon.com/blogs/containers/backup-and-restore-your-amazon-eks-cluster-resources-using-velero/

```sh
helm repo add vmware-tanzu https://vmware-tanzu.github.io/helm-charts
# create dedicated namespace
kubectl create namespace velero-system
```

```sh
helm upgrade --install velero vmware-tanzu/velero \
  --values infrastructure/helm/velero/values.yaml \
  --namespace velero-system
```

### MongoDB

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
```
#### On initial setup, let mongodb create it's secrets on it's own:
```sh
helm install mongodb bitnami/mongodb \
  --values infrastructure/helm/mongodb/values.yaml \
  --namespace core
```
> [!WARNING]
> Once mongodb is installed, the volumes persist throughout uninstalls or upgrades. The volumes retain the credentials used on creation. If you reinstall Mongodb or change credentials, you will have to delete the PVC and PV first.
> This issues manifests as "Authentication error" on the mongodb pods

#### For successive helm upgrades, pass the existing secrets
```sh
export MONGODB_REPLICA_SET_KEY=$(kubectl get secret --namespace "core" mongodb -o jsonpath="{.data.mongodb-replica-set-key}" | base64 -d)
export MONGODB_ROOT_PASSWORD=$(kubectl get secret --namespace "core" mongodb -o jsonpath="{.data.mongodb-root-password}" | base64 -d)

helm upgrade mongodb bitnami/mongodb \
  --values infrastructure/helm/mongodb/values.yaml \
  --set auth.rootPassword=$MONGODB_ROOT_PASSWORD \
  --set auth.replicaSetKey=$MONGODB_REPLICA_SET_KEY \
  --namespace core
```

#### Create base users in mongodb
```sh
# Create servicebind secrets for the default users
  for MONGO_USER in writer notifier collector
  do
    MONGO_PW=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 12`;
    KSECRET_NAME=mongodb-${MONGO_USER}-svcbind
    kubectl create secret generic $KSECRET_NAME \
      --namespace core \
      --type='servicebinding.io/mongodb' \
      --from-literal=type='mongodb' \
      --from-literal=provider='bitnami' \
      --from-literal=host='mongodb-headless.core.svc.cluster.local' \
      --from-literal=port='27017' \
      --from-literal=username="${MONGO_USER}"\
      --from-literal=password="$MONGO_PW" \
      --from-literal=uri="mongodb+srv://${MONGO_USER}:${MONGO_PW}@mongodb-headless.core.svc.cluster.local/?tls=false&ssl=false"
  done
```
```sh
# Run a mongodb client container. Then within the container execute the create user script.
# Credentials for the single users are extracted via kubectl from the corresponding secrets (see secrets.md for how to create them)
export MONGODB_ROOT_PASSWORD=$(kubectl get secret --namespace core mongodb -o jsonpath='{.data.mongodb-root-password}' | base64 -d)
kubectl run --namespace core mongodb-client --rm --tty -i --restart='Never' --env="MONGODB_ROOT_PASSWORD=$MONGODB_ROOT_PASSWORD" --image docker.io/bitnami/mongodb:6.0.9-debian-11-r5 --command -- \
mongosh 'mongodb+srv://mongodb-headless.core.svc.cluster.local/admin?tls=false' --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --eval "
	db.createUser(
	  {
	    user: '`kubectl get secret --namespace core mongodb-writer-svcbind -o jsonpath='{.data.username}' | base64 -d`',
	    pwd: '`kubectl get secret --namespace core mongodb-writer-svcbind -o jsonpath='{.data.password}' | base64 -d`',
	    roles: [ { role: 'readWriteAnyDatabase', db: 'admin' } ]
	  }
	);
	db.createUser(
	  {
	    user: '`kubectl get secret --namespace core mongodb-notifier-svcbind -o jsonpath='{.data.username}' | base64 -d`',
	    pwd: '`kubectl get secret --namespace core mongodb-notifier-svcbind -o jsonpath='{.data.password}' | base64 -d`',
	    roles: [ { role: 'readAnyDatabase', db: 'admin' }, { role: 'readWrite', db: 'admin' } ]
	  }
	);
	db.createUser(
	  {
	    user: '`kubectl get secret --namespace core mongodb-collector-svcbind -o jsonpath='{.data.username}' | base64 -d`',
	    pwd: '`kubectl get secret --namespace core mongodb-collector-svcbind -o jsonpath='{.data.password}' | base64 -d`',
	    roles: [ { role: 'readAnyDatabase', db: 'admin' } ]
	  }
	);
"
```

### RabbitMQ

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
```

#### Initial setup
```sh
helm install rabbitmq bitnami/rabbitmq \
  --values infrastructure/helm/rabbitmq/values.yaml \
  --namespace core
```

#### Upgrade existing deployment
```sh
export RABBITMQ_PASSWORD=$(kubectl get secret --namespace "core" rabbitmq -o jsonpath="{.data.rabbitmq-password}" | base64 -d)
export RABBITMQ_ERLANG_COOKIE=$(kubectl get secret --namespace "core" rabbitmq -o jsonpath="{.data.rabbitmq-erlang-cookie}" | base64 -d)

helm upgrade rabbitmq bitnami/rabbitmq \
  --values infrastructure/helm/rabbitmq/values.yaml \
  --set auth.password=$RABBITMQ_PASSWORD \
  --set auth.erlangCookie=$RABBITMQ_ERLANG_COOKIE \
  --namespace core
  
# ATTENTION: After an upgrade of a major rabbitmq version, it is necessary to enable all stable feature flags (the GUI will scream at you to do so)
# This can either be done via the GUI (Admin/Feature Flags), or by executing this:
kubectl exec -n core rabbitmq-0 --container rabbitmq -- rabbitmqctl enable_feature_flag all
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

**Reminder: authenticate to docker via `docker login ghcr.io` before invoking the `docker push` command.**

```sh
# note the label is there to link the package to the gh repo
docker build -t ghcr.io/noi-techpark/infrastructure-v2/notifier:latest infrastructure/notifier \
--label "org.opencontainers.image.source=https://github.com/noi-techpark/infrastructure-v2"

docker push ghcr.io/noi-techpark/infrastructure-v2/notifier:latest
```

```sh
helm upgrade --install notifier ./infrastructure/helm/notifier/notifier \
  --values infrastructure/helm/notifier/values.yaml \
  --namespace core
```

### Raw writer

```sh
# must be logged in on ghcr.io for docker push
(cd infrastructure/raw-writer; ./build.sh)
```

```sh
helm upgrade --install raw-writer ./infrastructure/helm/raw-writer/raw-writer \
  --values infrastructure/helm/raw-writer/values.yaml \
  --namespace core
```

### Nginx ingress
TODO: values.yaml contains hardcoded subnet of dev environment.

```sh
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
```

```sh
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
for NAMESPACE in core collector
do
  kubectl create --namespace $NAMESPACE -f infrastructure/ingress/cert-manager/letsencrypt-staging-clusterissuer.yaml
  kubectl create --namespace $NAMESPACE -f infrastructure/ingress/cert-manager/letsencrypt-prod-clusterissuer.yaml
done
```

### Secret reflector
To make secrets visible across namespaces we use kubernetes-reflector

```sh
helm repo add emberstack https://emberstack.github.io/helm-charts
```
```sh
helm upgrade --install reflector --namespace kube-system emberstack/reflector
```

## Open Data Hub core applications
These are the legacy core applications of the Open Data Hub
These use already existing containers and pipelines and are configured via env variables set as "env." values.

Image tags are hardcoded, so upgrading the applications has to be done by updating image tags (commit hash)

> [!IMPORTANT]
> The core applications rely on secrets in `secrets.md` for package repository and database access

### Ninja API
Outbound mobility API used to query mobility data
TODO: rewrite to /v2/ context and add the STA mobility proxies
```sh
helm upgrade --install ninja-api infrastructure/helm/ninja-api/ninja-api --namespace core --values infrastructure/helm/ninja-api/values.yaml
```
### Mobility core / writer
Endpoint where data collectors and elaborations push their mobility data. Also maintains and versions the mobility database schema
```sh
helm upgrade --install bdp-core infrastructure/helm/bdp-core/bdp-core --namespace core --values infrastructure/helm/bdp-core/values.yaml
```
### Analytics
Frontend application that uses Ninja-API to visualize mobility data on maps and charts
```sh
helm upgrade --install analytics infrastructure/helm/analytics/analytics --namespace core --values infrastructure/helm/analytics/values.yaml
```
### Tourism API
Tourism API
```sh
helm upgrade --install tourism-api infrastructure/helm/tourism-api/tourism-api --namespace core --values infrastructure/helm/tourism-api/values.yaml
```
### Tourism importer
Tourism importer
```sh
helm upgrade --install tourism-importer infrastructure/helm/tourism-importer/tourism-importer --namespace core --values infrastructure/helm/tourism-importer/values.yaml
```