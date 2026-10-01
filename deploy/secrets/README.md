<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# Secrets

`<env>.yaml` holds every credential of an environment, encrypted with sops for that
environment's age key: one key for test, another for prod, so the test key cannot read prod.
Public halves are in `/.sops.yaml`, private halves in the password manager; locally both go
into `~/.config/sops/age/keys.txt`, one after the other. The `cluster-secrets` release renders it into the cluster.

Filling a new environment, in this order (`deploy/preflight.sh <env>` first confirms the
tools and the age key are in place):

```sh
# 1. generated values; creates the file, never overwrites
deploy/secrets/init.sh deploy/secrets/<env>.yaml

# 2. everything Terraform created (infrastructure repo, terraform/scaleway/opendatahub-<env>)
deploy/secrets/from-terraform.py deploy/secrets/<env>.yaml ../infrastructure/terraform/scaleway/opendatahub-<env>

# 3. what neither can produce, by hand
sops deploy/secrets/<env>.yaml

# check: lists whatever is still empty
deploy/secrets/init.sh deploy/secrets/<env>.yaml
```

Nothing here prints a secret. To look at one: `sops decrypt --extract '["postgres"]["host"]' deploy/secrets/<env>.yaml`.

## sops and age in short

```sh
# key setup, once: paste the env keys from the password manager, one after the other
mkdir -p ~/.config/sops/age && install -m 600 /dev/null ~/.config/sops/age/keys.txt
$EDITOR ~/.config/sops/age/keys.txt

sops deploy/secrets/test.yaml                                        # edit in $EDITOR, re-encrypted on save
sops decrypt --extract '["rabbitmq"]["password"]' deploy/secrets/test.yaml   # read one value
echo '"new-value"' | sops set --value-stdin deploy/secrets/test.yaml '["rabbitmq"]["password"]'  # write one

# new key for an environment (e.g. someone with access leaves):
age-keygen -o new.txt                  # store it in the password manager, then delete new.txt
# put its public key into /.sops.yaml, then re-encrypt:
sops updatekeys deploy/secrets/test.yaml
```

After a new key, rotate the values themselves too: old versions in git stay readable with the old key.

## Where each value comes from

| Key | Source | Filled by |
|---|---|---|
| `mongodb.rootPassword`, `mongodb.replicaSetKey`, `mongodb.users.*` | random | `init.sh` |
| `rabbitmq.password`, `rabbitmq.erlangCookie` | random | `init.sh` |
| `basicAuth.{tempo,prometheus,rawDataBridge}.{password,htpasswd}` | random, htpasswd via `openssl passwd -apr1` | `init.sh` |
| `velero.accessKey`, `velero.secretKey` | Terraform `shared`, output `access_keys.velero` | `from-terraform.py` |
| `rawS3.bucketName`, `rawS3.accessKeyId`, `rawS3.secretAccessKey` | Terraform `shared`, outputs `buckets.raw`, `access_keys.raw` | `from-terraform.py` |
| `scalewayDns.accessKey`, `scalewayDns.secretKey` | Terraform `default/dns`, output `k8s_dns_opendatahub_<env>_access_key` | `from-terraform.py` |
| `postgres.host`, `postgres.port`, `postgres.readwritePassword`, `postgres.readonlyPassword` | Terraform `db`, outputs `endpoints.timeseries`, `credentials["timeseries/bdp"]`, `credentials["timeseries/bdp_readonly"]` | `from-terraform.py` |
| `tourism.pgconnection` | Terraform `db`, built from `endpoints.content` and `credentials["content/tourism"]` | `from-terraform.py` |
| `registry.readToken`, `registry.readWriteToken` | GitHub PATs of `noi-techpark-bot`, `read:packages` / `write:packages` | by hand |
| `oauthCollector.clientSecret` | Keycloak, realm `noi`, client `odh-mobility-datacollector` → Credentials | by hand |

Non-secret values that also come from Terraform live in `deploy/environments/<env>.yaml`:
`ingress.lbId` (`kubernetes` output `ingress_lb_id`) and `velero.bucket`
(`shared` output `buckets.velero`).

## Scaleway API keys must not expire

The Velero, raw data and DNS keys are used by the cluster unattended, so they are created
**without an expiry**. This requires the organization IAM security setting
`MaxAPIKeyExpirationDuration` to stay unlimited. With a limit, Scaleway gives new keys an
expiry, the keys stop working when it passes, and every Terraform plan shows them being
replaced.

## Migration from AWS

The test environment currently shares the registry tokens and the collector Keycloak client secret
with the AWS test cluster; they were copied with

```sh
deploy/secrets/from-aws-cluster.py deploy/secrets/test.yaml dev
```

Issue separate ones for Scaleway before the AWS cluster goes away, or revoking the AWS
credentials breaks this environment too.
