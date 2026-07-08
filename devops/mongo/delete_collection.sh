#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./delete_collection.sh <database> <collection> before <YYYY-MM-DD>
#   ./delete_collection.sh <database> <collection> all
#   ./delete_collection.sh <database> <collection> drop
#   ./delete_collection.sh <database> <collection> days <n>
#
# Deletes documents by the 'bsontimestamp' field (Date).
# 'drop' removes the collection entirely, returning space to the OS immediately (no compact needed).
# All other modes use deleteMany and require compact afterward to reclaim disk space.
NAMESPACE="core"
DB="${1:?database required}"
COLL="${2:?collection required}"
MODE="${3:?mode required: before <date> | all | drop | days <n>}"

RELEASE="mongodb"
SECRET_NAME="mongodb"
LABEL_SELECTOR="app.kubernetes.io/instance=${RELEASE},app.kubernetes.io/name=mongodb"

case "$MODE" in
  before)
    CUTOFF_DATE="${4:?cutoff date required (YYYY-MM-DD)}"
    FILTER="{ bsontimestamp: { \$lt: new Date(\"${CUTOFF_DATE}\") } }"
    DESCRIPTION="older than ${CUTOFF_DATE}"
    ;;
  all)
    FILTER="{}"
    DESCRIPTION="all documents"
    ;;
  drop)
    FILTER=""
    DESCRIPTION="entire collection (drop)"
    ;;
  days)
    N_DAYS="${4:?number of days to retain required}"
    FILTER="{ bsontimestamp: { \$lt: new Date(Date.now() - ${N_DAYS} * 86400 * 1000) } }"
    DESCRIPTION="older than ${N_DAYS} days"
    ;;
  *)
    echo "Unknown mode '${MODE}'. Use: before <date> | all | drop | days <n>" >&2
    exit 1
    ;;
esac

if ! kubectl auth can-i get pods -n "$NAMESPACE" &>/dev/null; then
  echo "Not authenticated to Kubernetes or missing permissions. Run 'kubectl get pods -n ${NAMESPACE}' to diagnose." >&2
  exit 1
fi

PASSWORD=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.mongodb-root-password}' | base64 -d)

mongo_eval() {
  local pod="$1"
  local js="$2"
  kubectl exec -n "$NAMESPACE" "$pod" -c mongodb -- \
    mongosh "mongodb://root:${PASSWORD}@localhost/?authSource=admin&directConnection=true" --quiet --eval "$js"
}

echo "Discovering pods..."
PODS=$(kubectl get pods -n "$NAMESPACE" -l "$LABEL_SELECTOR" -o jsonpath='{.items[*].metadata.name}' \
  | tr ' ' '\n' | grep -v 'arbiter' | tr '\n' ' ')

PRIMARY_POD=""
for pod in $PODS; do
  STATE=$(mongo_eval "$pod" 'rs.status().myState === 1 ? "PRIMARY" : "OTHER"')
  STATE=$(echo "$STATE" | tr -d '\r\n ')
  if [ "$STATE" = "PRIMARY" ]; then
    PRIMARY_POD="$pod"
    break
  fi
done

if [ -z "$PRIMARY_POD" ]; then
  echo "Could not determine primary pod. Aborting."
  exit 1
fi

echo "Primary: $PRIMARY_POD"

echo ""
echo "--- Collection diagnostics ---"
mongo_eval "$PRIMARY_POD" "
  const coll = db.getSiblingDB('${DB}')['${COLL}'];
  const total = coll.estimatedDocumentCount();
  print('Total documents (estimate): ' + total);
  if (total === 0) { print('Collection is empty or does not exist.'); }

  const sample = coll.findOne({}, { bsontimestamp: 1 });
  const tsField = sample && sample['bsontimestamp'];
  print('Sample bsontimestamp       : ' + JSON.stringify(tsField));
  print('Filter                     : ${DESCRIPTION}');
"
echo ""

echo "  WARNING: You are about to permanently delete documents from MongoDB."
echo ""
echo "    Namespace : ${NAMESPACE}"
echo "    Database  : ${DB}"
echo "    Collection: ${COLL}"
echo "    Scope     : ${DESCRIPTION}"
echo ""
echo "  This operation is IRREVERSIBLE. There is no undo."
echo ""
read -r -p "  Type 'yes' to confirm: " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
  echo "Aborted."
  exit 1
fi
echo ""

if [ "$MODE" = "drop" ]; then
  echo "Dropping ${DB}.${COLL}..."
  mongo_eval "$PRIMARY_POD" "db.getSiblingDB('${DB}')['${COLL}'].drop(); print('Dropped ${DB}.${COLL}');"
else
  JS=$(cat <<EOF
const db = db.getSiblingDB("${DB}");
const result = db["${COLL}"].deleteMany(${FILTER});
print("Deleted " + result.deletedCount + " documents (${DESCRIPTION}) from ${DB}.${COLL}");
EOF
  )
  echo "Deleting records in ${DB}.${COLL} (${DESCRIPTION})..."
  mongo_eval "$PRIMARY_POD" "$JS"
fi
