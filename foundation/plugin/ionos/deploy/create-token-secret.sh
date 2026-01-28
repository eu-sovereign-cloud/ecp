#!/usr/bin/env bash
set -euo pipefail

# Minimal helper to create the token credentials secret for the Upjet IONOS provider.
# Usage:
#   IONOS_TOKEN=xxxxx bash ./create-token-secret.sh
# Optional env:
#   SECRET_NAME=ionos-credentials
#   SECRET_NAMESPACE=crossplane-system

SECRET_NAME=${SECRET_NAME:-ionos-credentials}
SECRET_NAMESPACE=${SECRET_NAMESPACE:-crossplane-system}

# KUBECONFIG-only targeting for kubectl
KUBECTL_ARGS=${KUBECTL_ARGS:-}
if [[ -n "${KUBECONFIG:-}" ]]; then
  KUBECTL_ARGS+=" --kubeconfig ${KUBECONFIG}"
fi

if [[ -z "${IONOS_TOKEN:-}" ]]; then
  echo "Error: IONOS_TOKEN is not set. Export IONOS_TOKEN and retry."
  exit 1
fi

# Ensure namespace exists
kubectl ${KUBECTL_ARGS} create namespace "${SECRET_NAMESPACE}" --dry-run=client -o yaml | kubectl ${KUBECTL_ARGS} apply -f - >/dev/null

# Create/update secret idempotently with the expected key 'credentials'
CREDENTIALS_JSON="{\"token\":\"${IONOS_TOKEN}\"}"

kubectl ${KUBECTL_ARGS} -n "${SECRET_NAMESPACE}" create secret generic "${SECRET_NAME}" \
  --from-literal=credentials="${CREDENTIALS_JSON}" \
  --dry-run=client -o yaml | kubectl ${KUBECTL_ARGS} apply -f -

echo "Created/updated secret ${SECRET_NAME} in namespace ${SECRET_NAMESPACE}."