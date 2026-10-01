<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Helmfile

`deploy/helmfile.yaml.gotmpl` brings up the platform layer of a cluster (ingress,
cert-manager, databases, message brokers, core services, monitoring) in one command. It
replaces the step-by-step instructions in [helm.md](helm.md) for the Scaleway clusters.

Secrets live encrypted in `deploy/secrets/<env>.yaml` ([sops](https://github.com/getsops/sops)
with an [age](https://github.com/FiloSottile/age) key) and are rendered into the cluster by the
`cluster-secrets` release. Nothing secret is created by hand anymore.

## Layout

```
deploy/                       what runs where
  helmfile.yaml.gotmpl
  environments/<env>.yaml     non-secret per-environment settings
  secrets/<env>.yaml          sops-encrypted; init.sh creates and extends them
  values/<release>/
    values.yaml               any cloud, any environment
    scaleway.yaml             Scaleway specifics (storage classes, backup target)
    <env>.yaml                per environment (image tags, AWS-era hostnames)
    scaleway-<env>.yaml       Scaleway per environment (hostnames under scw.opendatahub.testingmachine.eu)
  aws/                        everything only the manual AWS setup in helm.md uses:
    values/<release>/         AWS-only releases, and aws.yaml / aws-<env>.yaml overlays
    manifests/                storage classes, Route53 issuers, pomerium kustomization
infrastructure/helm/          charts we author, nothing environment-specific
```

A release picks up `values.yaml`, `scaleway.yaml`, `<env>.yaml` and `scaleway-<env>.yaml`
in that order; each is optional. The helmfile never reads `deploy/aws/`; it goes away with
the AWS clusters, together with helm.md. After that, `scaleway.yaml` and
`scaleway-<env>.yaml` can be folded into `values.yaml` and `<env>.yaml`.

## Scaleway prerequisites

Created outside this repository, before the first apply:

All of it lives in the `infrastructure` repository, `terraform/scaleway/opendatahub-<env>`:

- `kubernetes`: the Kapsule cluster, tagged `scw-filestorage-csi` for File Storage
  (`sfs-standard`, used by `files`) on a POP2 pool. Autoscaling is set on the pool, and
  Kapsule runs its own metrics-server; neither is deployed from here. The ingress load balancer
  and its IP are created here too, output `ingress_lb_id` goes to `ingress.lbId` (see
  [Replacing the cluster](#replacing-the-cluster)).
- `shared`: the Velero and raw data buckets with one API key each. `velero.bucket` goes
  to the environment file, the rest to the sops file via `from-terraform.py`.
- `../default/dns`: the zone `scw.opendatahub.testingmachine.eu` (project Default), the
  records of the proxies, and the environment's DNS key for cert-manager and external-dns,
  copied by `from-terraform.py`.
- `proxy`: `proxy-opendatahub-test`, the Caddy in front of the cluster's ingress.
- `db`: the `timeseries` and `content` PostgreSQL instances, copied to the sops file via
  `from-terraform.py`.

## Tools and access

`deploy/preflight.sh <env>` checks all of the following and says what is missing:

- [helm](https://helm.sh/docs/intro/install/), [helmfile](https://helmfile.readthedocs.io/en/latest/#installation),
  [sops](https://github.com/getsops/sops/releases), [age](https://github.com/FiloSottile/age/releases), `openssl`
- helm plugins: `helmfile init` (installs helm-diff and helm-secrets)
- the age key of the environment (one for test, one for prod), from the password manager, in
  `~/.config/sops/age/keys.txt`; the file holds several keys, one after the other
- kube contexts named `dev-scw` / `scw-odh-prod`. Each environment is pinned to its
  context, so `-e prod` cannot be applied to the test cluster by mistake.

## Secrets

Every key, where it comes from and how to fill it: [deploy/secrets/README.md](../deploy/secrets/README.md).
In short: `init.sh` generates, `from-terraform.py` copies what Terraform created, the rest
is entered with `sops`.

Commit the encrypted file. To rotate a value, edit it with `sops` and apply again. MongoDB
users are updated by the `mongodb-users` job on every change.

When migrating an existing cluster, copy the current values into the file instead of
generating new ones: restored MongoDB volumes still expect the old credentials.

## Apply

```sh
deploy/preflight.sh test
cd deploy
helmfile -e test diff
helmfile -e test apply

# a single release
helmfile -e test apply -l name=raw-data-bridge
```

## Replacing the cluster

Deleting the cluster (`delete_additional_resources = true`) also deletes its volumes, so
MongoDB, RabbitMQ, Prometheus and `files` start empty: back up first. It deletes the load
balancers whose name starts with the cluster ID too, **with their IPs**, even reserved
ones. That is why the ingress load balancer and its IP belong to Terraform
(`kubernetes`, `scaleway_lb.ingress`): the cluster only configures its frontends and
backends (`scw-loadbalancer-externally-managed`), and both survive the cluster.

```sh
# 1. the controller removes its frontends and backends from the load balancer
cd deploy && helmfile -e test destroy -l name=ingress-nginx

# 2. new cluster and pool (infrastructure repo)
terraform -chdir=terraform/scaleway/opendatahub-test/kubernetes apply -replace=scaleway_k8s_cluster.main

# 3. kubeconfig of the new cluster under the same context name; the cluster ID is the
#    kubernetes_cluster_id output without its fr-par/ prefix
kubectl config delete-context dev-scw
scw k8s kubeconfig install <cluster-id> region=fr-par
kubectl config rename-context scw-main-eu-01-<cluster-id> dev-scw

# 4. everything else, in one run
deploy/preflight.sh test
cd deploy && helmfile -e test apply
```

The DNS records stay in the zone throughout; external-dns in the new cluster takes over the
ones it owns. The proxy needs no change.

### If step 1 was forgotten

The load balancer and its IP are intact, but the frontends and backends of the deleted
cluster are still on it. The new controller cannot create its own: `helmfile apply` waits
on ingress-nginx until it times out, its Service gets no external IP, and its events show
`failed creating frontend port: 80`. Remove the leftovers; the controller recreates them
for the new nodes within seconds, and the apply continues (or run it again):

```sh
LB=$(terraform -chdir=terraform/scaleway/opendatahub-test/kubernetes output -raw ingress_lb_id)
for f in $(scw lb frontend list zone=fr-par-1 lb-id=${LB#*/} -o json | jq -r '.[].id'); do
  scw lb frontend delete "$f" zone=fr-par-1
done
for b in $(scw lb backend list zone=fr-par-1 lb-id=${LB#*/} -o json | jq -r '.[].id'); do
  scw lb backend delete "$b" zone=fr-par-1
done
```

## Not covered yet

- pomerium: on AWS a kustomization (`deploy/aws/manifests/pomerium`, see [helm.md](helm.md#pomerium-ingress-protected-endpoints));
  on Scaleway it needs a chart, its own reserved IP and DNS records, and a Keycloak client
- notifier (no chart in the repository)
