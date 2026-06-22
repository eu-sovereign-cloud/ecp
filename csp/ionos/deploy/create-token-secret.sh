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

# Create/update secret idempotently.
# The secret manifest is built via stdin to avoid leaking the token in process
# arguments (visible via ps, CI logs, /proc/*/cmdline, etc.).
kubectl ${KUBECTL_ARGS} apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: ${SECRET_NAME}
  namespace: ${SECRET_NAMESPACE}
type: Opaque
stringData:
  credentials: '{"token":"${IONOS_TOKEN}"}'
EOF

echo "Created/updated secret ${SECRET_NAME} in namespace ${SECRET_NAMESPACE}."
