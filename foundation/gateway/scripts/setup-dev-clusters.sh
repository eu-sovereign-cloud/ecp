#!/bin/bash

set -euo pipefail

# --- Configuration ---
GLOBAL_CLUSTER_NAME="global"
REGIONAL_CLUSTER_NAME="regional"

# Kubeconfig files will be created in this directory
KUBECONFIG_DIR="${HOME}/.kube/multi-cluster-demo"
GLOBAL_KUBECONFIG_PATH="${KUBECONFIG_DIR}/global-config"
REGIONAL_KUBECONFIG_PATH="${KUBECONFIG_DIR}/regional-config"

# Resolve repository-relative paths (so script can be run from any cwd)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
# Delegator root (sibling of gateway) used for CRDs now generated under delegator/api
DELEGATOR_ROOT="$(cd "${GATEWAY_ROOT}/../delegator" && pwd)"
APIS_CRDS_DIR="${DELEGATOR_ROOT}/api/generated/crds"  # Updated to new location
CONFIG_SETUP_DIR="${GATEWAY_ROOT}/config/k8s-dev-setup"
REGIONAL_STORAGE_CONFIG_DIR="${CONFIG_SETUP_DIR}/regional/storage"

# Docker Images
GLOBAL_IMAGE="registry.secapi.cloud/global-server:latest"
REGIONAL_IMAGE="registry.secapi.cloud/regional-server:latest"

# Deployment and CRD files (updated to point to new apis folder & correct config dir)
GLOBAL_DEPLOYMENT_YAML="${CONFIG_SETUP_DIR}/global-deployment.yaml"
REGIONAL_DEPLOYMENT_YAML="${CONFIG_SETUP_DIR}/regional-deployment.yaml"
REGION_CRD_YAML="${APIS_CRDS_DIR}/regions/v1.secapi.cloud_regions.yaml"
REGIONS_RBAC_YAML="${CONFIG_SETUP_DIR}/global_regions_rbac.yaml"
# Storage SKU CRD & RBAC for regional cluster
STORAGE_SKU_CRD_YAML="${APIS_CRDS_DIR}/block-storage/storage.v1.secapi.cloud_storage-skus.yaml"
REGIONAL_STORAGE_SKU_CR="${REGIONAL_STORAGE_CONFIG_DIR}/storage-sku.yaml"
REGIONAL_STORAGE_SKU_RBAC_YAML="${REGIONAL_STORAGE_CONFIG_DIR}/regional-storage-sku-rbac.yaml"

# Verify required files exist early to fail fast
ensure_file() { if [ ! -f "$1" ]; then echo "Error: Required file not found: $1" >&2; exit 1; fi }
ensure_file "${GLOBAL_DEPLOYMENT_YAML}"
ensure_file "${REGIONAL_DEPLOYMENT_YAML}"
ensure_file "${REGION_CRD_YAML}"
ensure_file "${REGIONS_RBAC_YAML}"
ensure_file "${STORAGE_SKU_CRD_YAML}"
ensure_file "${REGIONAL_STORAGE_SKU_CR}"
ensure_file "${REGIONAL_STORAGE_SKU_RBAC_YAML}"

# Region Details
REGION_NAME="region"
REGION_PROVIDER="seca.compute"

# --- Helper Functions ---
check_command() {
  if ! command -v "$1" &> /dev/null; then
    echo "Error: '$1' is not installed. Please install it to continue." >&2
    exit 1
  fi
}

# --- Script ---

# 1. Check for dependencies
check_command "kind"
check_command "kubectl"
check_command "docker"

# 2. Create clusters with kind
echo "--- Step 1: Creating 'global' and 'regional' clusters with kind ---"
kind create cluster --name "${GLOBAL_CLUSTER_NAME}"
kind create cluster --name "${REGIONAL_CLUSTER_NAME}"

# 3. Save kubeconfig files
echo "--- Step 2: Saving kubeconfig files to '${KUBECONFIG_DIR}' ---"
mkdir -p "${KUBECONFIG_DIR}"
kind get kubeconfig --name "${GLOBAL_CLUSTER_NAME}" > "${GLOBAL_KUBECONFIG_PATH}"
kind get kubeconfig --name "${REGIONAL_CLUSTER_NAME}" > "${REGIONAL_KUBECONFIG_PATH}"
echo "Global kubeconfig saved to: ${GLOBAL_KUBECONFIG_PATH}"
echo "Regional kubeconfig saved to: ${REGIONAL_KUBECONFIG_PATH}"

# Wait for nodes to be ready
echo "--- Waiting for clusters to be ready ---"
echo "Waiting for global cluster nodes..."
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" wait --for=condition=Ready node --all --timeout=300s
echo "Global cluster is ready."

echo "Waiting for regional cluster nodes..."
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" wait --for=condition=Ready node --all --timeout=300s
echo "Regional cluster is ready."

# 4. Load local Docker images into the clusters
echo "--- Step 3: Loading local Docker images into clusters ---"
kind load docker-image "${GLOBAL_IMAGE}" --name "${GLOBAL_CLUSTER_NAME}"
kind load docker-image "${REGIONAL_IMAGE}" --name "${REGIONAL_CLUSTER_NAME}"

# 5. Apply Region CRD to the global cluster
echo "--- Step 4: Applying Region CRD to the global cluster ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${REGION_CRD_YAML}"

# 6. Apply RBAC to the global cluster
echo "--- Step 5: Applying RBAC configuration to the global cluster ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${REGIONS_RBAC_YAML}"

# 7. Apply StorageSKU CRD to the regional cluster
echo "--- Step 6: Applying StorageSKU CRD to the regional cluster ---"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -f "${STORAGE_SKU_CRD_YAML}"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -f "${REGIONAL_STORAGE_SKU_CR}"
# Wait until CRD is established to avoid race conditions with controllers
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" wait --for=condition=Established crd/storage-skus.storage.v1.secapi.cloud --timeout=60s || {
  echo "Warning: StorageSKU CRD establishment wait timed out" >&2
}

# 8. Apply StorageSKU RBAC to the regional cluster
echo "--- Step 7: Applying StorageSKU RBAC to the regional cluster ---"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -f "${REGIONAL_STORAGE_SKU_RBAC_YAML}"

# 9. Apply deployments to clusters
echo "--- Step 8: Applying deployments ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${GLOBAL_DEPLOYMENT_YAML}"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -f "${REGIONAL_DEPLOYMENT_YAML}"
echo "Deployments applied. Waiting for services to be ready..."
sleep 15 # Give services and endpoints time to initialize

# 10. Discover the regional service endpoint
echo "--- Step 9: Discovering regional service endpoint ---"
# For kind, the node's IP is the IP of its control-plane Docker container
NODE_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${REGIONAL_CLUSTER_NAME}-control-plane")
if [ -z "$NODE_IP" ]; then
  echo "Error: Could not determine Node IP for the regional cluster." >&2
  exit 1
fi
echo "Discovered Regional Node IP: ${NODE_IP}"

NODE_PORT=$(kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" get svc regional-server-svc -n default -o jsonpath='{.spec.ports[0].nodePort}')
if [ -z "$NODE_PORT" ]; then
  echo "Error: Could not determine NodePort for regional-server-svc." >&2
  exit 1
fi
echo "Discovered Regional Service NodePort: ${NODE_PORT}"

REGIONAL_API_ENDPOINT="http://${NODE_IP}:${NODE_PORT}"
echo "Constructed Regional API Endpoint: ${REGIONAL_API_ENDPOINT}"

# 11. Generate and apply the Region CR to the global cluster
echo "--- Step 10: Registering regional cluster in the global cluster ---"
cat <<EOF | kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f -
apiVersion: v1.secapi.cloud/v1
kind: Region
metadata:
  name: "${REGION_NAME}"
spec:
  availableZones:
    - "${REGION_NAME}-a"
    - "${REGION_NAME}-b"
  providers:
    - name: "${REGION_PROVIDER}"
      url: "${REGIONAL_API_ENDPOINT}"
      version: "v1"
EOF

echo "--- Setup Complete! ---"
echo "To interact with the clusters, use:"
echo "export KUBECONFIG=${GLOBAL_KUBECONFIG_PATH}"
echo "kubectl get all"
echo ""
echo "export KUBECONFIG=${REGIONAL_KUBECONFIG_PATH}"
echo "kubectl get all"
echo ""
echo "To access the APIs locally via port-forwarding, run the following commands in separate terminals:"
echo "# Forward Global API to localhost:8080"
echo "kubectl --kubeconfig ${GLOBAL_KUBECONFIG_PATH} port-forward deployment/global-api-deployment 8080:8080"
echo ""
echo "# Forward Regional API to localhost:8081"
echo "kubectl --kubeconfig ${REGIONAL_KUBECONFIG_PATH} port-forward svc/regional-server-svc 8081:80"
echo ""
echo "To delete the clusters, run: kind delete cluster --name ${GLOBAL_CLUSTER_NAME} && kind delete cluster --name ${REGIONAL_CLUSTER_NAME}"