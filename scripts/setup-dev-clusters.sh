#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
GLOBAL_CLUSTER_NAME="global"
REGIONAL_CLUSTER_NAME="regional"

# Kubeconfig files will be created in this directory
KUBECONFIG_DIR="${HOME}/.kube/multi-cluster-demo"
GLOBAL_KUBECONFIG_PATH="${KUBECONFIG_DIR}/global-config"
REGIONAL_KUBECONFIG_PATH="${KUBECONFIG_DIR}/regional-config"

# Docker Images
GLOBAL_IMAGE="global-server:latest"
REGIONAL_IMAGE="regional-server:latest"

# Deployment and CRD files
GLOBAL_DEPLOYMENT_YAML="config/k8s-dev-setup/global-deployment.yaml"
REGIONAL_DEPLOYMENT_YAML="config/k8s-dev-setup/regional-deployment.yaml"
REGION_CRD_YAML="apis/generated/regions/regions.secapi.eu.io_regions.yaml" # Assumes you have the CRD definition here
RBAC_YAML="config/k8s-dev-setup/rbac.yaml"

# Region Details
REGION_NAME="region1"
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

# 4. Load local Docker images into the clusters
echo "--- Step 3: Loading local Docker images into clusters ---"
kind load docker-image "${GLOBAL_IMAGE}" --name "${GLOBAL_CLUSTER_NAME}"
kind load docker-image "${REGIONAL_IMAGE}" --name "${REGIONAL_CLUSTER_NAME}"

# 5. Apply CRD to the global cluster
echo "--- Step 4: Applying Region CRD to the global cluster ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${REGION_CRD_YAML}"

# 6. Apply RBAC to the global cluster
echo "--- Step 5: Applying RBAC configuration to the global cluster ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${RBAC_YAML}"

# 6. Apply deployments to clusters
echo "--- Step 5: Applying deployments ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${GLOBAL_DEPLOYMENT_YAML}"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -f "${REGIONAL_DEPLOYMENT_YAML}"
echo "Deployments applied. Waiting for services to be ready..."
sleep 15 # Give services and endpoints time to initialize

# 7. Discover the regional service endpoint
echo "--- Step 6: Discovering regional service endpoint ---"
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

# 8. Generate and apply the Region CR to the global cluster
echo "--- Step 7: Registering regional cluster in the global cluster ---"
cat <<EOF | kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f -
apiVersion: regions.secapi.eu.io/v1
kind: Regions
metadata:
  name: "${REGION_NAME}"
  namespace: default
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