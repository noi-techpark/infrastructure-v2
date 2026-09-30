#!/usr/bin/env bash
# Checks that this machine can deploy an environment: tools, helm plugins, the sops age
# key and the environment's kube context. Changes nothing.
# usage: deploy/preflight.sh <test|prod>
set -uo pipefail

env=${1:?usage: $0 <test|prod>}
root=$(cd "$(dirname "$0")/.." && pwd)
secrets="$root/deploy/secrets/$env.yaml"
failed=0

if [ -t 1 ]; then green=$'\033[32m' yellow=$'\033[33m' red=$'\033[31m' reset=$'\033[0m'; else green= yellow= red= reset=; fi
ok()   { printf '  %sok%s    %s\n' "$green" "$reset" "$*"; }
warn() { printf '  %swarn%s  %s\n' "$yellow" "$reset" "$*"; }
fail() { printf '  %sFAIL%s  %s\n' "$red" "$reset" "$*"; failed=1; }

# version_at_least <found> <minimum>: dotted numeric comparison
version_at_least() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" = "$2" ]; }

# tool <name> <minimum version or -> <install hint>
tool() {
  local name=$1 min=$2 hint=$3 version
  if ! command -v "$name" >/dev/null; then fail "$name not found; $hint"; return; fi
  version=$("$name" "${@:4}" 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
  if [ "$min" != - ] && ! version_at_least "$version" "$min"; then
    fail "$name $version, need $min or newer; $hint"
  else
    ok "$name $version"
  fi
}

echo "tools"
tool helm     3.14 "https://helm.sh/docs/intro/install/"                   version --short
tool helmfile 1.0  "https://github.com/helmfile/helmfile/releases"          --version
tool sops     3.10 "https://github.com/getsops/sops/releases"               --version
tool age      -    "https://github.com/FiloSottile/age/releases"            --version
tool kubectl  -    "https://kubernetes.io/docs/tasks/tools/"                version --client
tool openssl  -    "system package (used by secrets/init.sh)"              version
tool python3  3.8  "system package (used by the secrets scripts)"          --version
if command -v terraform >/dev/null; then
  ok "terraform $(terraform version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1) (only for secrets/from-terraform.py)"
else
  warn "terraform not found; only needed for secrets/from-terraform.py"
fi

echo "helm plugins"
for plugin in diff secrets; do
  if helm plugin list 2>/dev/null | awk 'NR>1 {print $1}' | grep -qx "$plugin"; then
    ok "$plugin"
  else
    fail "$plugin missing; run: helmfile init"
  fi
done

echo "sops age key"
default_file=${XDG_CONFIG_HOME:-$HOME/.config}/sops/age/keys.txt
# The environment's own key: from the file's sops metadata, or from its .sops.yaml rule
# before the file exists. Another environment's key must not count.
if [ -f "$secrets" ]; then
  recipients=$(grep -oE 'recipient: age1[0-9a-z]+' "$secrets" | awk '{print $2}' | sort -u)
else
  recipients=$(grep -A1 -F "deploy/secrets/$env" "$root/.sops.yaml" | grep -oE 'age1[0-9a-z]+' | sort -u)
fi
# sops reads every one of these sources, so any of them may hold the key.
match=""
found=0
check_source() { # <label> <public keys, one per line>
  [ -n "$2" ] || return
  found=1
  if grep -qxFf <(printf '%s\n' "$recipients") <<<"$2"; then match=$1; fi
}
if [ -n "${SOPS_AGE_KEY:-}" ]; then
  check_source SOPS_AGE_KEY "$(printf '%s\n' "$SOPS_AGE_KEY" | age-keygen -y 2>/dev/null)"
fi
for file in ${SOPS_AGE_KEY_FILE:-} "$default_file"; do
  [ -f "$file" ] || continue
  check_source "$file" "$(age-keygen -y "$file" 2>/dev/null)"
  if [ "$(stat -c %a "$file")" != 600 ]; then warn "$file is readable by others; chmod 600 it"; fi
done
if [ "$found" = 0 ]; then
  fail "no age key (checked SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, $default_file); get the $env key from the password manager"
elif [ -z "$match" ]; then
  fail "age keys found, but not the $env key; get it from the password manager"
else
  ok "$env key found in $match"
fi
if [ ! -f "$secrets" ]; then
  fail "deploy/secrets/$env.yaml missing; create it with: deploy/secrets/init.sh deploy/secrets/$env.yaml"
elif sops decrypt "$secrets" >/dev/null 2>&1; then
  ok "deploy/secrets/$env.yaml decrypts"
else
  fail "deploy/secrets/$env.yaml does not decrypt with the available keys"
fi

echo "cluster"
context=$(awk -v env="$env" '
  $0 ~ "^  " env ":$" { inside = 1; next }
  inside && /^  [^ ]/ { inside = 0 }
  inside && $1 == "kubeContext:" { print $2; exit }
' "$root/deploy/helmfile.yaml.gotmpl")
if [ -z "$context" ]; then
  fail "no kubeContext for environment $env in deploy/helmfile.yaml.gotmpl"
elif ! kubectl config get-contexts -o name 2>/dev/null | grep -qx "$context"; then
  fail "kube context $context not configured; see the cluster README in the infrastructure repo"
elif ! kubectl --context "$context" --request-timeout=10s get --raw /readyz >/dev/null 2>&1; then
  fail "kube context $context configured, but the cluster does not answer"
else
  ok "kube context $context reachable"
fi

echo
if [ "$failed" = 0 ]; then
  echo "ready to deploy $env: cd deploy && helmfile -e $env diff"
else
  echo "not ready; fix the FAIL lines above"
  exit 1
fi
