#!/usr/bin/env bash
# Fills in the keys missing from an environment's sops file. Existing values are never
# touched, so it is safe to rerun after adding a key here.
# usage: deploy/secrets/init.sh deploy/secrets/test.yaml
set -euo pipefail

file=${1:?usage: $0 <secrets file>}
manual=()

if [[ ! -f $file ]]; then
  echo 'placeholder: ""' | sops encrypt --filename-override "$file" /dev/stdin > "$file"
  sops unset "$file" '["placeholder"]'
fi

rand() {
  local s
  s=$(openssl rand -base64 96)
  s=${s//[^A-Za-z0-9]/}
  printf %s "${s:0:${1:-24}}"
}

get() { sops decrypt --extract "$1" "$file" 2>/dev/null; }
put() { sops set "$file" "$1" "\"$2\""; }

# ensure KEY [LENGTH]: random value if missing
ensure() {
  [[ -n $(get "$1") ]] || put "$1" "$(rand "${2:-24}")"
}

# ensure_basic_auth NAME: password plus the matching htpasswd line for user "operator"
ensure_basic_auth() {
  local key="[\"basicAuth\"][\"$1\"]" pw
  [[ -n $(get "$key[\"password\"]") ]] && return
  pw=$(rand 24)
  put "$key[\"password\"]" "$pw"
  put "$key[\"htpasswd\"]" "operator:$(openssl passwd -apr1 "$pw")"
}

# expect KEY: cannot be generated; leaves an empty value and reports it
expect() {
  [[ -n $(get "$1") ]] && return
  get "$1" >/dev/null || put "$1" ""
  manual+=("$1")
}

ensure '["mongodb"]["rootPassword"]'
ensure '["mongodb"]["replicaSetKey"]' 64
for user in writer notifier collector prometheus; do
  ensure "[\"mongodb\"][\"users\"][\"$user\"]"
done

ensure '["rabbitmq"]["password"]'
ensure '["rabbitmq"]["erlangCookie"]' 32

for name in tempo prometheus rawDataBridge; do
  ensure_basic_auth "$name"
done

expect '["registry"]["readToken"]'
expect '["registry"]["readWriteToken"]'
expect '["postgres"]["host"]'
expect '["postgres"]["port"]'
expect '["postgres"]["readwritePassword"]'
expect '["postgres"]["readonlyPassword"]'
expect '["rawS3"]["bucketName"]'
expect '["rawS3"]["accessKeyId"]'
expect '["rawS3"]["secretAccessKey"]'
expect '["velero"]["accessKey"]'
expect '["velero"]["secretKey"]'
expect '["scalewayDns"]["accessKey"]'
expect '["scalewayDns"]["secretKey"]'
expect '["oauthCollector"]["clientSecret"]'
expect '["ninjaApi"]["oauthClientSecret"]'

if (( ${#manual[@]} )); then
  echo "Still empty, fill in with: sops $file"
  printf '  %s\n' "${manual[@]}"
fi
