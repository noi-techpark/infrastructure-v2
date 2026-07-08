#!/usr/bin/env bash
set -euo pipefail

# Usage: ./compact_after_delete.sh <database> <collection>
NAMESPACE="core"
DB="${1:?database required}"
COLL="${2:?collection required}"

RELEASE="mongodb"
SECRET_NAME="mongodb"
LABEL_SELECTOR="app.kubernetes.io/instance=${RELEASE},app.kubernetes.io/name=mongodb"
POLL_INTERVAL=15       # seconds between compact progress checks
LAG_THRESHOLD_SEC=5    # max replication lag before starting compact on stepped-down primary

if ! kubectl auth can-i get pods -n "$NAMESPACE" &>/dev/null; then
  echo "Not authenticated to Kubernetes or missing permissions. Run 'kubectl get pods -n ${NAMESPACE}' to diagnose." >&2
  exit 1
fi

PASSWORD=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.mongodb-root-password}' | base64 -d)

mongo_eval() {
  local pod="$1"
  local js="$2"
  kubectl exec -n "$NAMESPACE" "$pod" -c mongodb -- \
    mongosh "mongodb://root:${PASSWORD}@localhost/?authSource=admin&directConnection=true&readPreference=secondaryPreferred" --quiet --eval "$js"
}

reclaimable_bytes() {
  local pod="$1"
  mongo_eval "$pod" "
    const s = db.getSiblingDB('${DB}')['${COLL}'].stats();
    const wt = s.wiredTiger && s.wiredTiger['block-manager'];
    print(wt ? Number(wt['file bytes available for reuse']) : -1);
  " | tr -d '\r\n '
}

running_compact_opid() {
  local pod="$1"
  mongo_eval "$pod" "
    const op = db.currentOp({ 'command.compact': '${COLL}' });
    print(op.inprog.length > 0 ? op.inprog[0].opid : '');
  " | tr -d '\r\n '
}

# Waits until the pod's replication lag is below LAG_THRESHOLD_SEC.
wait_for_replication() {
  local pod="$1"
  echo "    Waiting for ${pod} to catch up on replication (threshold: ${LAG_THRESHOLD_SEC}s)..."
  while true; do
    local lag
    lag=$(mongo_eval "$pod" "
      const s = rs.status();
      const self = s.members.find(m => m.self);
      const primary = s.members.find(m => m.state === 1);
      if (!self || !primary) { print(-1); }
      else { print(Math.abs(primary.optimeDate - self.optimeDate) / 1000); }
    " | tr -d '\r\n ')
    if [ -z "$lag" ] || [ "$lag" = "-1" ]; then
      echo "    Could not determine lag, retrying..."
      sleep "$POLL_INTERVAL"
      continue
    fi
    local lag_int=${lag%.*}
    echo "    Replication lag: ${lag}s"
    if [ "$lag_int" -le "$LAG_THRESHOLD_SEC" ]; then
      echo "    Caught up."
      break
    fi
    sleep "$POLL_INTERVAL"
  done
}

wait_for_rstl() {
  local pod="$1"
  echo "    Waiting for RSTL contention to clear on ${pod}..."
  while true; do
    local contention
    contention=$(mongo_eval "$pod" "
      const ops = db.currentOp({ waitingForLock: true }).inprog
        .filter(op => op.lockStats && op.lockStats.ReplicationStateTransition);
      print(ops.length);
    " | tr -d '\r\n ')
    if [ "${contention:-1}" = "0" ]; then
      echo "    RSTL clear."
      break
    fi
    echo "    ${contention} op(s) waiting on RSTL, retrying in ${POLL_INTERVAL}s..."
    sleep "$POLL_INTERVAL"
  done
}

compact_pod() {
  local pod="$1"
  local wait_replication="${2:-false}"

  if [ "$wait_replication" = "true" ]; then
    wait_for_replication "$pod"
    wait_for_rstl "$pod"
  fi

  local opid
  opid=$(running_compact_opid "$pod")
  if [ -n "$opid" ]; then
    echo ">>> Compact already running on ${pod} (opid=${opid}), waiting for it to finish..."
  else
    local reuse
    reuse=$(reclaimable_bytes "$pod")
    local reuse_mb=$(( reuse / 1024 / 1024 ))
    echo ">>> Starting compact of ${DB}.${COLL} on ${pod} (${reuse_mb} MB reclaimable)..."
    kubectl exec -n "$NAMESPACE" "$pod" -c mongodb -- bash -c "
      nohup mongosh -u root -p '${PASSWORD}' --authenticationDatabase admin --quiet \
        --eval \"db.getSiblingDB('${DB}').runCommand({compact: '${COLL}', force: true})\" \
        > /tmp/compact_${COLL}.log 2>&1 &
      echo \$!
    "
  fi

  echo "    Polling every ${POLL_INTERVAL}s..."
  while true; do
    sleep "$POLL_INTERVAL"
    opid=$(running_compact_opid "$pod" 2>/dev/null) || { echo "    kubectl exec failed, retrying..."; continue; }
    if [ -n "$opid" ]; then
      echo "    Still running on ${pod} (opid=${opid})..."
    else
      echo ">>> Compact finished on ${pod}."
      break
    fi
  done

  local after
  after=$(reclaimable_bytes "$pod" 2>/dev/null) || after=-1
  if [ "$after" -ge 0 ] 2>/dev/null; then
    echo "    Reclaimable after compact: $(( after / 1024 / 1024 )) MB"
  fi
}

echo "Discovering pods..."
PODS=$(kubectl get pods -n "$NAMESPACE" -l "$LABEL_SELECTOR" -o jsonpath='{.items[*].metadata.name}' \
  | tr ' ' '\n' | grep -v 'arbiter' | tr '\n' ' ')
echo "Pods: $PODS"

PRIMARY_POD=""
SECONDARY_PODS=()
for pod in $PODS; do
  STATE=$(mongo_eval "$pod" 'const s = rs.status().myState; s === 1 ? "PRIMARY" : s === 2 ? "SECONDARY" : s === 7 ? "ARBITER" : "OTHER"')
  STATE=$(echo "$STATE" | tr -d '\r\n ')
  echo "Pod $pod is $STATE"
  if [ "$STATE" = "PRIMARY" ]; then
    PRIMARY_POD="$pod"
  elif [ "$STATE" = "SECONDARY" ]; then
    SECONDARY_PODS+=("$pod")
  fi
done

if [ -z "$PRIMARY_POD" ]; then
  echo "Could not determine primary pod. Aborting."
  exit 1
fi

echo "Primary: $PRIMARY_POD"
echo "Secondaries: ${SECONDARY_PODS[*]}"
echo ""
echo "  Choose compact mode:"
echo ""
echo "  [1] Non-blocking  — preferred for active collections. May cause brief service disruptions due to primary switch."
echo "  [2] Blocking      — preferred for inactive collections. Reads and writes to ${DB}.${COLL} are blocked for the duration."
echo ""
read -r -p "  Enter 1 or 2: " MODE_CHOICE
echo ""

case "$MODE_CHOICE" in
  1)
    for pod in "${SECONDARY_PODS[@]}"; do
      compact_pod "$pod"
    done

    echo ">>> Stepping down primary ${PRIMARY_POD}..."
    mongo_eval "$PRIMARY_POD" 'rs.stepDown(60)' || true

    kubectl wait pod "$PRIMARY_POD" -n "$NAMESPACE" --for=condition=Ready --timeout=120s

    compact_pod "$PRIMARY_POD" true
    ;;
  2)
    echo ">>> Compacting ${DB}.${COLL} directly on primary ${PRIMARY_POD} (reads/writes to this collection are blocked)..."
    compact_pod "$PRIMARY_POD"
    ;;
  *)
    echo "Invalid choice. Aborting." >&2
    exit 1
    ;;
esac

echo "Done."
