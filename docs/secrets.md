<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Common secrets

## ghcr container registry secrets
docker registry access to pull private packages.
In our case it's ghcr, but could be anything
Create these in in github as a personal access token (classic) with package:read/write permissions accordingly

Both a read/write and read permission secret (write currently needed by camel-k, shouldn't in the future)

```sh
kubectl create secret docker-registry container-registry-rw \
  --docker-server=ghcr.io \
  --docker-email=digital@noi.bz.it \
  --docker-username=noi-techpark-bot \
  --docker-password=[TOKEN] \
  --namespace core

kubectl create secret docker-registry container-registry-r \
  --docker-server=ghcr.io \
  --docker-email=digital@noi.bz.it \
  --docker-username=noi-techpark-bot \
  --docker-password=[TOKEN] \
  --namespace core
  
# AFTER (RE)CREATING DON'T FORGET TO ADD REFLECTOR ANNOTATIONS, SEE BELOW
```

## rabbitmq
A default servicebind secret is created by rabbitmq helm chart

## mongodb
See `helm.md`

## postgresql
Create secrets for rw and ro users using servicebind standard

```sh
  # terraform init so you have access to the state
  (cd infrastructure/terraform/db; terraform init)

  # Extract the user credentials and database coordinates from terraform:
  EXTRACTED_JSON=`terraform -chdir=infrastructure/terraform/db output -json | jq -r '{hostname: .odh_postgres_hostname.value, port: .odh_postgres_port.value, pw_readwrite: .odh_postgres_password_bdp.value, pw_readonly: .odh_postgres_password_bdp_readonly.value}'`

  # Get the value from the extracted json. You can supply your values in another way if you didn't use the terraform script
  POSTGRES_HOST=`jq '.hostname' -r <<< "$EXTRACTED_JSON"`
  POSTGRES_PORT=`jq '.port' -r <<< "$EXTRACTED_JSON"`
  POSTGRES_R_PW=`jq '.pw_readonly' -r <<< "$EXTRACTED_JSON"`
  POSTGRES_RW_PW=`jq '.pw_readwrite' -r <<< "$EXTRACTED_JSON"`
  POSTGRES_DB=bdp
  POSTGRES_SCHEMAS='intimev2,public'

  kubectl create secret generic postgres-readwrite-svcbind \
    --namespace core \
    --type='servicebinding.io/postgresql' \
    --from-literal=type='postgresql' \
    --from-literal=provider='rds' \
    --from-literal=host="$POSTGRES_HOST" \
    --from-literal=port="$POSTGRES_PORT" \
    --from-literal=username="bdp" \
    --from-literal=password="$POSTGRES_RW_PW" \
    --from-literal=db="$POSTGRES_DB" \
    --from-literal=schema="$POSTGRES_SCHEMAS" \
    --from-literal=uri="postgresql://bdp:${POSTGRES_RW_PW}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?currentSchema=${POSTGRES_SCHEMAS}"

  kubectl create secret generic postgres-read-svcbind \
    --namespace core \
    --type='servicebinding.io/postgresql' \
    --from-literal=type='postgresql' \
    --from-literal=provider='rds' \
    --from-literal=host="$POSTGRES_HOST" \
    --from-literal=port="$POSTGRES_PORT" \
    --from-literal=username="bdp_readonly" \
    --from-literal=password="$POSTGRES_R_PW" \
    --from-literal=db="$POSTGRES_DB" \
    --from-literal=schema="$POSTGRES_SCHEMAS" \
    --from-literal=uri="postgresql://bdp_readonly:${POSTGRES_R_PW}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?currentSchema=${POSTGRES_SCHEMAS}"
```

## Tourism test DB
Only for initial tests, we use the existing tourism DB.
Format
```sh
  kubectl create secret generic tourism \
    --namespace core \
    --from-literal=pgconnection="Server=hostname.domain.com;Port=5432;User ID=******;Password=********;Database=tourism"
```

## Oauth data collectors
Used by data collectors to authenticate themselves (mostly for pushing to writer API)
```bash
  kubectl create secret generic oauth-collector \
    --namespace collector \
    --from-literal=authorizationUri='https://auth.opendatahub.testingmachine.eu/auth' \
    --from-literal=tokenUri='https://auth.opendatahub.testingmachine.eu/auth/realms/noi/protocol/openid-connect/token' \
    --from-literal=clientId='odh-mobility-datacollector' \
    --from-literal=clientSecret='*************'
```

# Expose secrets to other namespaces
Secrets are namespace-bound in Kubernetes.  
To make certain secrets available across namespaces, we use kubernetes-reflector.
```sh
  kubectl --namespace core annotate secret \
    rabbitmq-svcbind mongodb-collector-svcbind container-registry-r \
    reflector.v1.k8s.emberstack.com/reflection-allowed='true' \
    reflector.v1.k8s.emberstack.com/reflection-auto-enabled='true' \
    reflector.v1.k8s.emberstack.com/reflection-allowed-namespaces='collector'
```
