#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_FILE="${DIR}/provider.yaml"
PROVIDER_PKG="${PROVIDER_PKG:-}"
PROVIDER_NAME="${PROVIDER_NAME:-}"

# ProviderConfig/Secret defaults (can be overridden)
PROVIDER_CONFIG_NAME="${PROVIDER_CONFIG_NAME:-cluster-ionos-provider-config}"
PROVIDER_CONFIG_SECRET="${PROVIDER_CONFIG_SECRET:-ionos-credentials}"
PROVIDER_CONFIG_SECRET_NS="${PROVIDER_CONFIG_SECRET_NS:-crossplane-system}"

# KUBECONFIG-only targeting for kubectl
KUBECTL_ARGS=${KUBECTL_ARGS:-}
if [[ -n "${KUBECONFIG:-}" ]]; then
  KUBECTL_ARGS+=" --kubeconfig ${KUBECONFIG}"
fi

# Build the manifest to apply, allowing override via PROVIDER_PKG/PROVIDER_NAME
apply_input=""
if [[ -n "${PROVIDER_PKG}" ]]; then
  apply_input=$(cat <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: ${PROVIDER_NAME:-provider-upjet-ionoscloud}
spec:
  package: ${PROVIDER_PKG}
EOF
)
else
  if [[ ! -f "${INSTALL_FILE}" ]]; then
    echo "Error: ${INSTALL_FILE} not found. Ensure the file exists in the same directory as this script."
    exit 1
  fi
  apply_input="$(cat "${INSTALL_FILE}")"
fi

# Ensure the Provider CRD is installed (wait briefly if Crossplane is still starting)
CRD_NAME="providers.pkg.crossplane.io"
if ! kubectl ${KUBECTL_ARGS} get crd "${CRD_NAME}" >/dev/null 2>&1; then
  echo "Provider CRD ${CRD_NAME} not found. Waiting up to 120s for Crossplane to finish installing..."
  for i in {1..24}; do
    if kubectl ${KUBECTL_ARGS} get crd "${CRD_NAME}" >/dev/null 2>&1; then
      break
    fi
    sleep 5
  done
fi

if ! kubectl ${KUBECTL_ARGS} get crd "${CRD_NAME}" >/dev/null 2>&1; then
  echo "Provider CRD not found. Please install Crossplane first by running 'make install-crossplane'."
  exit 1
fi

echo "Applying Provider package..."
echo "${apply_input}" | kubectl ${KUBECTL_ARGS} apply -f -

echo "Provider CR applied."

echo "Waiting for Provider to become healthy (this may take a minute)..."

# Determine the provider name for wait loop
if [[ -n "${PROVIDER_NAME}" ]]; then
  provider_name="${PROVIDER_NAME}"
else
  provider_name=$(echo "${apply_input}" | awk '/^  name:/ {print $2; exit}')
  if [[ -z "${provider_name}" ]]; then provider_name="provider-upjet-ionoscloud"; fi
fi

# Wait until the Provider has a Healthy condition with status True (best-effort)
# We'll wait up to 5 minutes.
timeout=300
interval=5
elapsed=0
while true; do
  if kubectl ${KUBECTL_ARGS} get provider "${provider_name}" -o jsonpath='{.status.conditions[?(@.type=="Healthy")].status}' 2>/dev/null | grep -q True; then
    echo "Provider ${provider_name} reports Healthy=true"
    break
  fi
  if kubectl ${KUBECTL_ARGS} get provider "${provider_name}" -o jsonpath='{.status.conditions[0].reason}' 2>/dev/null | grep -qi "installed"; then
    echo "Provider ${provider_name} installation appears complete"
    break
  fi
  if [[ ${elapsed} -ge ${timeout} ]]; then
    echo "Timed out waiting for Provider ${provider_name} to become healthy. Check 'kubectl get provider ${provider_name} -o yaml' for details."
    exit 1
  fi
  sleep ${interval}
  elapsed=$((elapsed + interval))
done

echo "Provider ${provider_name} installed/ready"

# Now create/apply the ProviderConfig that references the secret with the token
# Allow user to override secret/name/namespace via PROVIDER_CONFIG_* env vars

echo "Ensuring ProviderConfig and referenced Secret (secret: ${PROVIDER_CONFIG_SECRET} in namespace ${PROVIDER_CONFIG_SECRET_NS})..."

# If the secret doesn't exist, abort — this script will not create credentials for you.
if kubectl ${KUBECTL_ARGS} get secret -n "${PROVIDER_CONFIG_SECRET_NS}" "${PROVIDER_CONFIG_SECRET}" >/dev/null 2>&1; then
  echo "Secret ${PROVIDER_CONFIG_SECRET} already exists in namespace ${PROVIDER_CONFIG_SECRET_NS}."
else
  echo "Secret ${PROVIDER_CONFIG_SECRET} not found in namespace ${PROVIDER_CONFIG_SECRET_NS}."
  echo "Please create it before running this script, for example:"
  echo "  IONOS_TOKEN=<your-token> bash ${DIR}/create-token-secret.sh"
  exit 1
fi

# Build ProviderConfig YAML only (do not create the secret here)
providerconfig_yaml=$(cat <<EOF
apiVersion: upjet-ionoscloud.m.ionoscloud.io/v1beta1
kind: ClusterProviderConfig
metadata:
#  namespace: ${PROVIDER_CONFIG_SECRET_NS}
  name: ${PROVIDER_CONFIG_NAME}
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: ${PROVIDER_CONFIG_SECRET_NS}
      name: ${PROVIDER_CONFIG_SECRET}
      key: credentials
EOF
)

# Apply ProviderConfig
echo "Applying ClusterProviderConfig ${PROVIDER_CONFIG_NAME}"
apply_output=$(echo "${providerconfig_yaml}" | kubectl ${KUBECTL_ARGS} apply -f - 2>&1 || true)
# Print the raw kubectl apply output for transparency
echo "${apply_output}"

echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} applied (or already present)."

# Fast-path: check cluster-scoped resources right away.
if kubectl ${KUBECTL_ARGS} get clusterproviderconfig "${PROVIDER_CONFIG_NAME}" >/dev/null 2>&1; then
  echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} confirmed (immediate)."
  exit 0
fi

# Fallback: retry loop to handle eventual consistency in discovery
verify_timeout=30
verify_interval=2
verify_elapsed=0
found=false
while [[ ${verify_elapsed} -lt ${verify_timeout} ]]; do
  # Try group-qualified get
  if kubectl ${KUBECTL_ARGS} get clusterproviderconfig.upjet-ionoscloud.m.ionoscloud.io "${PROVIDER_CONFIG_NAME}" >/dev/null 2>&1; then
    echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} confirmed (group-qualified)."
    found=true
    break
  fi
  # Try simple get
  if kubectl ${KUBECTL_ARGS} get clusterproviderconfig "${PROVIDER_CONFIG_NAME}" >/dev/null 2>&1; then
    echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} confirmed."
    found=true
    break
  fi
  # Try generic listing
  if kubectl ${KUBECTL_ARGS} get clusterproviderconfig -o name 2>/dev/null | grep -q "/${PROVIDER_CONFIG_NAME}$"; then
    echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} confirmed (listed)."
    found=true
    break
  fi
  sleep ${verify_interval}
  verify_elapsed=$((verify_elapsed + verify_interval))
done

if [[ "${found}" != "true" ]]; then
  echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} not visible after ${verify_timeout}s. Doing a last resort search..."
  if kubectl ${KUBECTL_ARGS} get clusterproviderconfig -o name 2>/dev/null | grep -q "/${PROVIDER_CONFIG_NAME}$"; then
    echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} was found. Listing matches:"
    kubectl ${KUBECTL_ARGS} get clusterproviderconfig -o name 2>/dev/null | grep "/${PROVIDER_CONFIG_NAME}$" || true
    found=true
  else
    echo "ClusterProviderConfig ${PROVIDER_CONFIG_NAME} could not be confirmed. You can inspect cluster resources with: kubectl get clusterproviderconfig | grep ${PROVIDER_CONFIG_NAME}"
  fi
fi

if [[ "${found}" == "true" ]]; then
  echo "ProviderConfig creation verified."
else
  echo "Warning: ProviderConfig ${PROVIDER_CONFIG_NAME} not found. Continuing, but installation may not be complete."
fi

exit 0
