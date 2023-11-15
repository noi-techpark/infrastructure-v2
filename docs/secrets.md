# Common secrets

## ghcr container registry secrets
docker registry access to pull private packages.
In our case it's ghcr, but could be anything
Create these in in github as a personal access token (classic) with package:read/write permissions accordingly

Both a read/write and read permission secret (write currently needed by camel-k, shouldn't in the future)

```sh
kubectl create secret docker-registry container-registry-rw \
  --docker-username=[USER] \
  --docker-password=[TOKEN] \
  --namespace core

kubectl create secret docker-registry container-registry-r \
  --docker-username=[USER] \
  --docker-password=[TOKEN] \
  --namespace core
```

## rabbitmq
A default servicebind secret is created by rabbitmq helm chart

## mongodb
Create secrets with random password to be used by services to access mongo.
NB: You also have to create the user in mongo with these credentials. see helm.md for that


```sh
  for MONGO_USER in writer notifier collector
  do
    MONGO_PW=`head /dev/urandom | tr -dc A-Za-z0-9 | head -c 12`;
    kubectl create secret generic mongodb-${MONGO_USER}-svcbind \
      --namespace core \
      --type='servicebinding.io/mongodb' \
      --from-literal=type='mongodb' \
      --from-literal=provider='bitnami' \
      --from-literal=host='mongodb-headless.core.svc.cluster.local' \
      --from-literal=port='27017' \
      --from-literal=username="${MONGO_USER}"\
      --from-literal=password="$MONGO_PW" \
      --from-literal=uri="mongodb+srv://${MONGO_USER}:${MONGO_PW}@mongodb-headless.core.svc.cluster.local?tls=false"
  done
```
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
    --from-literal=uri="postgresql://bdp_readonly:${POSTGRES_R_PW}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?currentSchema=${POSTGRES_SCHEMAS}"
```