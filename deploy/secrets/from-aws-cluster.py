#!/usr/bin/env python3
# SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0

# Migration only: copies the credentials that exist nowhere but in the AWS cluster
# (registry tokens, the collector Keycloak client secret) into an environment's sops file, without
# printing them. Replace them with values issued for Scaleway before AWS is shut down.
# usage: deploy/secrets/from-aws-cluster.py deploy/secrets/test.yaml <kube context>
import base64, json, subprocess, sys

if len(sys.argv) != 3:
    sys.exit("usage: deploy/secrets/from-aws-cluster.py <sops file> <kube context>")
secrets, context = sys.argv[1], sys.argv[2]


def secret(namespace, name):
    out = subprocess.run(["kubectl", "--context", context, "get", "secret", "-n", namespace, name, "-o", "json"],
                         capture_output=True, text=True, check=True).stdout
    return {k: base64.b64decode(v).decode() for k, v in json.loads(out)["data"].items()}


def ghcr_token(name):
    auths = json.loads(secret("core", name)[".dockerconfigjson"])["auths"]
    entry = next(v for k, v in auths.items() if "ghcr.io" in k)
    return entry.get("password") or base64.b64decode(entry["auth"]).decode().split(":", 1)[1]


values = {
    '["registry"]["readToken"]': ghcr_token("container-registry-r"),
    '["registry"]["readWriteToken"]': ghcr_token("container-registry-rw"),
    '["oauthCollector"]["clientSecret"]': secret("collector", "oauth-collector")["clientSecret"],
}
for path, value in values.items():
    subprocess.run(["sops", "set", "--value-stdin", secrets, path],
                   input=json.dumps(value), text=True, check=True, stdout=subprocess.DEVNULL)
    print(f"set {path:36} len={len(value)}")
