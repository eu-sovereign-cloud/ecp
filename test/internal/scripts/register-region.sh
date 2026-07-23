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
# Must match the extraPortMappings entry in internal/kind-config/regional-cluster.yaml,
# which is what makes this port reachable from the host.
REGIONAL_NODE_PORT="${MULTICLUSTER_REGIONAL_NODE_PORT:-30080}"
# The advertised host. A KIND node's own bridge IP works on a native Linux host
# but not from WSL2 against Docker Desktop, where the containers are in another
# VM's network namespace; the published host port works in both.
ADVERTISE_HOST="${MULTICLUSTER_ADVERTISE_HOST:-127.0.0.1}"

echo "--- Registering region '${REGION_NAME}' (${REGIONAL_CONTEXT} -> ${GLOBAL_CONTEXT}) ---"

# The suite reaches the regional gateway at whatever the Region CR advertises, so
# that address has to be routable from outside the regional cluster. ClusterIP —
# what the single-cluster stack deploys — is not, so the multicluster overlay
# deploys the regional gateway as a NodePort with a pinned port (see the
# kind-multicluster-stack target and gateway-regional/multicluster-values.yaml).
# Here we only read that port back and confirm the pin, so the advertised host
# port lines up with what the chart exposed and what regional-cluster.yaml
# publishes — an auto-assigned port would not be reachable from the host.
GOT_PORT=$(kubectl --context "${REGIONAL_CONTEXT}" -n "${SYSTEM_NAMESPACE}" \
    get svc "${GATEWAY_REGIONAL_SVC}" -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || true)
if [ "${GOT_PORT}" != "${REGIONAL_NODE_PORT}" ]; then
    echo "Error: ${GATEWAY_REGIONAL_SVC} nodePort is '${GOT_PORT}', expected ${REGIONAL_NODE_PORT}." >&2
    echo "Deploy the regional gateway with the multicluster overlay (make kind-multicluster-stack)," >&2
    echo "and create the regional cluster with internal/kind-config/regional-cluster.yaml." >&2
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
