#!/bin/bash
# Register the regional cluster in the global cluster.
#
# This is the join step of the two-cluster topology: the regional gateway is
# exposed on a NodePort, its externally reachable address is discovered, and a
# Region CR advertising that address is written to the GLOBAL cluster. Nothing
# else tells the global control plane where the region lives — clients find the
# regional API only by reading the provider URLs off this CR, which is exactly
# what test/e2e/multicluster asserts.
set -eo pipefail

source "$(dirname "$0")/common.sh"

setup_env
source_config

SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-e2e-ecp}"
# Must match the REGION env of the regional gateway deployment, otherwise it
# serves a region nobody advertises.
REGION_NAME="${MULTICLUSTER_REGION:-itbg-bergamo}"
GLOBAL_CONTEXT="${MULTICLUSTER_GLOBAL_CONTEXT:-kind-e2e-global}"
REGIONAL_CONTEXT="${MULTICLUSTER_REGIONAL_CONTEXT:-kind-e2e-regional}"
# Must match the extraPortMappings entry in internal/kind/regional-cluster.yaml,
# which is what makes this port reachable from the host.
REGIONAL_NODE_PORT="${MULTICLUSTER_REGIONAL_NODE_PORT:-30080}"
# The advertised host. A KIND node's own bridge IP works on a native Linux host
# but not from WSL2 against Docker Desktop, where the containers are in another
# VM's network namespace; the published host port works in both.
ADVERTISE_HOST="${MULTICLUSTER_ADVERTISE_HOST:-127.0.0.1}"

echo "--- Registering region '${REGION_NAME}' (${REGIONAL_CONTEXT} -> ${GLOBAL_CONTEXT}) ---"

# The suite reaches the regional gateway at whatever the Region CR advertises, so
# that address has to be routable from outside the regional cluster. ClusterIP —
# what the single-cluster stack deploys — is not. Patching rather than shipping a
# second manifest keeps the single-cluster deployment untouched. The nodePort is
# pinned rather than auto-assigned so it lines up with the published host port.
kubectl --context "${REGIONAL_CONTEXT}" -n "${SYSTEM_NAMESPACE}" patch svc gateway-regional-svc \
    --type merge \
    -p "{\"spec\":{\"type\":\"NodePort\",\"ports\":[{\"port\":80,\"targetPort\":8080,\"protocol\":\"TCP\",\"nodePort\":${REGIONAL_NODE_PORT}}]}}" >/dev/null

# Confirm the pin took: an auto-assigned port would not be published to the host.
GOT_PORT=$(kubectl --context "${REGIONAL_CONTEXT}" -n "${SYSTEM_NAMESPACE}" \
    get svc gateway-regional-svc -o jsonpath='{.spec.ports[0].nodePort}')
if [ "${GOT_PORT}" != "${REGIONAL_NODE_PORT}" ]; then
    echo "Error: gateway-regional-svc nodePort is ${GOT_PORT}, expected ${REGIONAL_NODE_PORT}" >&2
    echo "The regional cluster must be created with internal/kind/regional-cluster.yaml." >&2
    exit 1
fi

REGIONAL_URL="http://${ADVERTISE_HOST}:${REGIONAL_NODE_PORT}"
echo "Regional gateway reachable at ${REGIONAL_URL}"

# Overwrites the static test-data Region of the same name, whose in-cluster DNS
# URLs are meaningless across a cluster boundary.
kubectl --context "${GLOBAL_CONTEXT}" apply -f - <<EOF
apiVersion: v1.secapi.cloud/v1
kind: Region
metadata:
  name: "${REGION_NAME}"
spec:
  availableZones:
    - "${REGION_NAME}-a"
    - "${REGION_NAME}-b"
  providers:
    - name: "seca.workspace"
      url: "${REGIONAL_URL}/providers/seca.workspace"
      version: "v1"
    - name: "seca.storage"
      url: "${REGIONAL_URL}/providers/seca.storage"
      version: "v1"
    - name: "seca.network"
      url: "${REGIONAL_URL}/providers/seca.network"
      version: "v1"
    - name: "seca.compute"
      url: "${REGIONAL_URL}/providers/seca.compute"
      version: "v1"
EOF

echo "Region '${REGION_NAME}' registered."
