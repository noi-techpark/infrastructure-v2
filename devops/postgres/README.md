# Postgresql client pod

Creates a persistent postgresql client for long running queries or migrations

create with
```sh
# Create pod
kubectl apply -f postgres-client.yml -n core
# get a shell
kubectl exec --stdin --tty -n core postgres-client -- /bin/bash

# Use screen to have a persistent terminal session on the pod even when disconnecting
screen bash
# To detach from the session use Ctrl+A D
# To re-attach to running session:
screen -x

# Use psql to interactively run queries:
psql postgresql://${db}:${password}@${host}:$port

# Or scripts 
psql postgresql://${db}:${password}@${host}:$port -f script.sql -v ON_ERROR_STOP=1
```
Since the postgresql-readwrite secret is mounted directly to env, you can use that to connect