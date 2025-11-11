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
# observability stack
kubectl create namespace monitoring
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

### Amazon GP3 Storage class


```sh
kubectl apply -f infrastructure/helm/aws-storage-class/gp3.yaml
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
# Create servicebind secrets for the default users
  for MONGO_USER in prometheus
  do
    MONGO_PW=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 12`;
    KSECRET_NAME=mongodb-${MONGO_USER}-svcbind
    kubectl create secret generic $KSECRET_NAME \
      --namespace monitoring \
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
  db.createUser(
	  {
	    user: '`kubectl get secret --namespace monitoring mongodb-prometheus-svcbind -o jsonpath='{.data.username}' | base64 -d`',
	    pwd: '`kubectl get secret --namespace monitoring mongodb-prometheus-svcbind -o jsonpath='{.data.password}' | base64 -d`',
	    roles: [ { role:'clusterMonitor', db:'admin' }, { role:'read', db:'local' } ]
	  }
	);
"
```

### RabbitMQ

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
```

### Grafana tempo gateway secret
1. Install `htpasswd`

```sh
PASSWORD=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 24`;

# Generate the htpasswd string
HTPASSWD=$(htpasswd -nb operator "$PASSWORD")

# Create the Kubernetes secret with the generated values
kubectl create secret generic tempo-gateway-basic-auth -n monitoring \
  --from-literal=.htpasswd="$HTPASSWD" \
  --from-literal=.username="operator" \
  --from-literal=.password="$PASSWORD"
```

### Prometheus secret
1. Install `htpasswd`

```sh
PASSWORD=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 24`;

# Generate the htpasswd string
HTPASSWD=$(htpasswd -nb operator "$PASSWORD")

# Create the Kubernetes secret with the generated values
kubectl create secret generic prometheus-basic-auth -n monitoring \
  --from-literal=auth="$HTPASSWD" \
  --from-literal=.username="operator" \
  --from-literal=.password="$PASSWORD"
```

### Raw Data Bridge secret
1. Install `htpasswd`

```sh
PASSWORD=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 24`;

# Generate the htpasswd string
HTPASSWD=$(htpasswd -nb operator "$PASSWORD")

# Create the Kubernetes secret with the generated values
kubectl create secret generic raw-data-bridge-basic-auth -n core \
  --from-literal=auth="$HTPASSWD" \
  --from-literal=.username="operator" \
  --from-literal=.password="$PASSWORD"
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

### Router

```sh
# must be logged in on ghcr.io for docker push
(cd infrastructure/router; ./build.sh)
```

```sh
helm upgrade --install router ./infrastructure/helm/router/router \
  --values infrastructure/helm/router/values.yaml \
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

### Raw Data Bridge

```sh
# must be logged in on ghcr.io for docker push
(cd infrastructure/raw-data-bridge; ./build.sh)
```

```sh
helm upgrade --install raw-data-bridge ./infrastructure/helm/raw-data-bridge/raw-data-bridge \
  --values infrastructure/helm/raw-data-bridge/values.yaml \
  --values infrastructure/helm/raw-data-bridge/values.dev.yaml \
  --namespace core
```

**PROD**
```sh
helm upgrade --install raw-data-bridge ./infrastructure/helm/raw-data-bridge/raw-data-bridge \
  --values infrastructure/helm/raw-data-bridge/values.yaml \
  --values infrastructure/helm/raw-data-bridge/values.prod.yaml \
  --namespace core
```

### Nginx ingress

```sh
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
```

```sh
# Testing:
helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx  --namespace ingress-nginx --create-namespace \
--values infrastructure/helm/nginx-ingress/values.test.yaml 

# Production:
helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx  --namespace ingress-nginx --create-namespace \
--values infrastructure/helm/nginx-ingress/values.prod.yaml 
```

### Certmanager (https certificates)

```sh
helm repo add jetstack https://charts.jetstack.io
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.12.4 \
  --set installCRDs=true \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=<ROLE CREATED WITH TERRAFORM>

kubectl rollout restart deployment cert-manager -n cert-manager

  
# Create the letsencrypt issuers. 
# TODO: create a route53 issuer so we can use dns instead of http challenges
for NAMESPACE in core collector monitoring
do
  kubectl create --namespace $NAMESPACE -f infrastructure/ingress/cert-manager/letsencrypt-staging-clusterissuer.yaml
  kubectl create --namespace $NAMESPACE -f infrastructure/ingress/cert-manager/letsencrypt-prod-clusterissuer.yaml
done

# cluster dns issuers
kubectl create -f infrastructure/ingress/cert-manager/letsencrypt-dns-prod-clusterissuer.yaml
kubectl create -f infrastructure/ingress/cert-manager/letsencrypt-dns-staging-clusterissuer.yaml
```

### Pomerium Ingress (protected endpoints)

```
helm repo add pomerium https://helm.pomerium.io
helm repo update
```

Start pomerium proxy and wait for lb creation & certificate issue

```
export EIP_ALLOCATION=<PRIVATE IP ALLOCATED USING TERRAFORM HERE>
export ISSUER=letsencrypt-dns-prod
export AUTHENTICATE_URL=authenticate.internal.opendatahub.com
export IDP_URL=https://auth.opendatahub.com/auth/realms/noi

envsubst < infrastructure/helm/pomerium/service-patch.template.yaml > infrastructure/helm/pomerium/service-patch.yaml
envsubst < infrastructure/helm/pomerium/certificate.template.yaml > infrastructure/helm/pomerium/certificate.yaml
envsubst < infrastructure/helm/pomerium/pomerium.template.yaml > infrastructure/helm/pomerium/pomerium.yaml


kubectl apply -k infrastructure/helm/pomerium
kubectl apply -f infrastructure/helm/pomerium/pomerium.yaml

kubectl -n pomerium patch secret idp-secret --type='merge' -p '{"stringData":{"client_id":"<CLIENT HERE>","client_secret":"<SECRET HERE>"}}'
```

TO DELETE
```
kubectl delete -k infrastructure/helm/pomerium
```

keycloak side we need to setup a claim mapper of type "user role mapper" which maps the roles to a "top level claim", like "roles".
This because the default behaviour (putting roles in [client-id].roles) nested object is not supported by pomerium.


roles:

raw-data-bridge:read
rabbitmq:read

Configuration syntax
```
    annotations:
      cert-manager.io/issuer: "letsencrypt-private-prod"
      ingress.pomerium.io/pass_identity_headers: 'true'
      # ingress.pomerium.io/allow_any_authenticated_user: 'true'
      ingress.pomerium.io/policy: |
        allow:
          and:
            # - claim/protected-area_roles: "[test, admin]"
            - claim/roles_array: admin

    ingressClassName: pomerium
    tls:
      - hosts:
          - prometheus.dev.testingmachine.eu
        secretName: tls-prometheus
    hosts:
      - prometheus.dev.testingmachine.eu
    paths: 
      - "/"
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
Relies on external proxy to do SSL currently
```sh
helm upgrade --install --namespace core ninja-api  \
https://github.com/noi-techpark/it.bz.opendatahub.api.mobility-ninja/raw/refs/heads/main/infrastructure/helm/ninja-api-2.0.0.tgz \
--values https://raw.githubusercontent.com/noi-techpark/it.bz.opendatahub.api.mobility-ninja/refs/heads/main/infrastructure/helm/test.yaml
```
### Mobility core / writer
Endpoint where data collectors and elaborations push their mobility data. Also maintains and versions the mobility database schema
Relies on external proxy to do SSL currently
```sh
helm upgrade --install --namespace core bdp-core  \
https://github.com/noi-techpark/bdp-core/raw/refs/heads/main/infrastructure/helm/bdp-core-2.0.0.tgz \
--values https://raw.githubusercontent.com/noi-techpark/bdp-core/refs/heads/main/infrastructure/helm/test.yaml \
--set-string env.LOG_APPLICATION_VERSION=helm-setup \
--set-string oauth.clientSecret=************
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