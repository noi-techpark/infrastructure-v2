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
  Kapsule runs its own metrics-server; neither is deployed from here. Output `ingress_ip_id` goes to `loadBalancers.ingressIpId`.
- `shared`: the Velero and raw data buckets with one API key each, the DNS zone
  `scw.opendatahub.testingmachine.eu` and the cert-manager DNS01 key. `velero.bucket` goes
  to the environment file, the rest to the sops file via `from-terraform.py`.
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

## Not covered yet

- pomerium: on AWS a kustomization (`deploy/aws/manifests/pomerium`, see [helm.md](helm.md#pomerium-ingress-protected-endpoints));
  on Scaleway it needs a chart, its own reserved IP and DNS records, and a Keycloak client
- notifier (no chart in the repository)
