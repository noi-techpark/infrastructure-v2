#!/usr/bin/env python3
# SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

# Copies the credentials Terraform creates for an environment into its sops file,
# without printing them.
# usage: deploy/secrets/from-terraform.py deploy/secrets/test.yaml \
#          ../infrastructure/terraform/scaleway/opendatahub-test
# Each module's .env (Terraform and state bucket credentials) is loaded if present.
import json, os, subprocess, sys

if len(sys.argv) != 3:
    sys.exit("usage: deploy/secrets/from-terraform.py <sops file> <terraform/scaleway/opendatahub-ENV dir>")
secrets, tf_root = sys.argv[1], sys.argv[2]


def outputs(module):
    path = os.path.join(tf_root, module)
    env = dict(os.environ)
    dotenv = os.path.join(path, ".env")
    if os.path.exists(dotenv):
        loaded = subprocess.run(["bash", "-c", f"set -a; . {dotenv!r}; env -0"],
                                capture_output=True, text=True, check=True).stdout
        env.update(line.split("=", 1) for line in loaded.split("\0") if "=" in line)
    out = subprocess.run(["terraform", f"-chdir={path}", "output", "-json"],
                         capture_output=True, text=True, check=True, env=env).stdout
    return {k: v["value"] for k, v in json.loads(out).items()}


shared, db = outputs("shared"), outputs("db")
keys, endpoints, users = shared["access_keys"], db["endpoints"], db["credentials"]
content, tourism = endpoints["content"], users["content/tourism"]

# The cluster-secrets chart assumes these names.
assert endpoints["timeseries"]["database"] == "bdp"
assert users["timeseries/bdp"]["username"] == "bdp"
assert users["timeseries/bdp_readonly"]["username"] == "bdp_readonly"

values = {
    '["velero"]["accessKey"]': keys["velero"]["access_key"],
    '["velero"]["secretKey"]': keys["velero"]["secret_key"],
    '["rawS3"]["bucketName"]': shared["buckets"]["raw"],
    '["rawS3"]["accessKeyId"]': keys["raw"]["access_key"],
    '["rawS3"]["secretAccessKey"]': keys["raw"]["secret_key"],
    '["scalewayDns"]["accessKey"]': shared["cert_manager_access_key"]["access_key"],
    '["scalewayDns"]["secretKey"]': shared["cert_manager_access_key"]["secret_key"],
    '["postgres"]["host"]': endpoints["timeseries"]["host"],
    '["postgres"]["port"]': str(endpoints["timeseries"]["port"]),
    '["postgres"]["readwritePassword"]': users["timeseries/bdp"]["password"],
    '["postgres"]["readonlyPassword"]': users["timeseries/bdp_readonly"]["password"],
    '["tourism"]["pgconnection"]': (
        f'Server={content["host"]};Port={content["port"]};User ID={tourism["username"]};'
        f'Password={tourism["password"]};Database={content["database"]}'
    ),
}
for path, value in values.items():
    subprocess.run(["sops", "set", "--value-stdin", secrets, path],
                   input=json.dumps(value), text=True, check=True, stdout=subprocess.DEVNULL)
    print(f"set {path:36} len={len(value)}")
