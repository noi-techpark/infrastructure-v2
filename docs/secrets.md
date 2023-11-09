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
      --from-literal=uri="mongodb://${MONGO_USER}:${TMP_MONGO_PW}@mongodb-headless.core.svc.cluster.local"
  done
```