#!/bin/bash

set -euo pipefail

# --- Configuration ---
GLOBAL_CLUSTER_NAME="global"
REGIONAL_CLUSTER_NAME="regional"

# Increase etcd quota to avoid: etcdserver: mvcc: database space exceeded
# Default etcd quota is typically ~2GiB; bumping helps dev clusters with lots of CRDs/resources/events.
ETCD_QUOTA_BACKEND_BYTES="${ETCD_QUOTA_BACKEND_BYTES:-8589934592}" # 8GiB

# Kubeconfig files will be created in this directory
KUBECONFIG_DIR="${HOME}/.kube/multi-cluster-demo"
GLOBAL_KUBECONFIG_PATH="${KUBECONFIG_DIR}/global-config"
REGIONAL_KUBECONFIG_PATH="${KUBECONFIG_DIR}/regional-config"

# Resolve repository-relative paths (so script can be run from any cwd)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
API_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
API_CRDS_DIR="${API_ROOT}/charts/ecp/crds"
CONFIG_SETUP_DIR="${GATEWAY_ROOT}/config/k8s-dev-setup"
REGIONAL_CONFIG_DIR="${CONFIG_SETUP_DIR}/regional"

# Docker Images
GLOBAL_IMAGE="registry.secapi.cloud/global-server:latest"
REGIONAL_IMAGE="registry.secapi.cloud/regional-server:latest"

# Deployment and RBAC files
GLOBAL_DEPLOYMENT_YAML="${CONFIG_SETUP_DIR}/global-deployment.yaml"
REGIONAL_DEPLOYMENT_YAML="${CONFIG_SETUP_DIR}/regional-deployment.yaml"
REGIONS_RBAC_YAML="${CONFIG_SETUP_DIR}/global_regions_rbac.yaml"
REGIONAL_RBAC_YAML="${CONFIG_SETUP_DIR}/regional_rbac.yaml"

# Verify required files/directories exist early to fail fast
ensure_file() { if [ ! -f "$1" ]; then echo "Error: Required file not found: $1" >&2; exit 1; fi; }
ensure_dir() { if [ ! -d "$1" ]; then echo "Error: Required directory not found: $1" >&2; exit 1; fi; }
ensure_file "${GLOBAL_DEPLOYMENT_YAML}"
ensure_file "${REGIONAL_DEPLOYMENT_YAML}"
ensure_file "${REGIONS_RBAC_YAML}"
ensure_file "${REGIONAL_RBAC_YAML}"
ensure_dir "${API_CRDS_DIR}"
ensure_dir "${REGIONAL_CONFIG_DIR}"

# Region Details
REGION_NAME="region"

# --- Helper Functions ---
check_command() {
  if ! command -v "$1" &> /dev/null; then
    echo "Error: '$1' is not installed. Please install it to continue." >&2
    exit 1
  fi
}

wait_for_http_ok() {
  # Poll a URL until it returns any HTTP response (curl -f only cares that
  # the server answered, not the status code — readiness endpoints may 200
  # or 503 depending on backend wiring, but a refused/timed-out connection
  # means the Service/Endpoints aren't routable yet).
  local url="$1" label="$2" attempts="${3:-30}" pause_s="${4:-2}"
  echo "Waiting for ${label} API to answer at ${url} ..."
  local i
  for ((i = 1; i <= attempts; i++)); do
    if curl -s -o /dev/null --max-time 3 "${url}"; then
      echo "${label} API answering (attempt ${i}/${attempts})"
      return 0
    fi
    sleep "${pause_s}"
  done
  echo "Error: ${label} API at ${url} did not answer after $((attempts * pause_s))s" >&2
  return 1
}

create_kind_cluster() {
  local name="$1"

  # Create a temporary kind config that increases etcd quota.
  # kind uses kubeadm under the hood; this patch sets etcd local extra args.
  local kind_cfg
  kind_cfg="$(mktemp)"
  trap 'rm -f "${kind_cfg}"' RETURN

  cat >"${kind_cfg}" <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    etcd:
      local:
        extraArgs:
          quota-backend-bytes: "${ETCD_QUOTA_BACKEND_BYTES}"
EOF

  echo "Creating kind cluster '${name}' with etcd quota-backend-bytes=${ETCD_QUOTA_BACKEND_BYTES}"
  kind create cluster --name "${name}" --config "${kind_cfg}"
}

# --- Script ---

# 1. Check for dependencies
check_command "kind"
check_command "kubectl"
check_command "docker"
check_command "curl"

"${SCRIPT_DIR}/../../ci/scripts/kind-cgroup-preflight.sh"

# 2. Create clusters with kind
echo "--- Step 1: Creating 'global' and 'regional' clusters with kind ---"
create_kind_cluster "${GLOBAL_CLUSTER_NAME}"
create_kind_cluster "${REGIONAL_CLUSTER_NAME}"

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

# 5. Apply all CRDs from API_CRDS_DIR to both clusters
echo "--- Step 4: Applying all CRDs from ${API_CRDS_DIR} ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -R -f "${API_CRDS_DIR}"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -R -f "${API_CRDS_DIR}"

# Wait for all CRDs to be established
echo "Waiting for CRDs to be established..."
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" wait --for=condition=Established crd --all --timeout=60s || {
  echo "Warning: Some global CRDs may not be established" >&2
}
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" wait --for=condition=Established crd --all --timeout=60s || {
  echo "Warning: Some regional CRDs may not be established" >&2
}

# 6. Apply RBAC to the global cluster
echo "--- Step 5: Applying RBAC configuration to the global cluster ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${REGIONS_RBAC_YAML}"

# 7. Apply regional CRs and RBAC (storage, workspace, etc.)
echo "--- Step 6: Applying regional CRs and RBAC ---"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -f "${REGIONAL_RBAC_YAML}"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -R -f "${REGIONAL_CONFIG_DIR}"

# 8. Apply deployments to clusters
echo "--- Step 7: Applying deployments ---"
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" apply -f "${GLOBAL_DEPLOYMENT_YAML}"
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" apply -f "${REGIONAL_DEPLOYMENT_YAML}"
echo "Deployments applied. Waiting for rollout..."
kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" rollout status deployment/global-api-deployment --timeout=180s
kubectl --kubeconfig "${REGIONAL_KUBECONFIG_PATH}" rollout status deployment/regional-server --timeout=180s

# 9. Discover the regional service endpoint
echo "--- Step 8: Discovering regional service endpoint ---"
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

# rollout status only confirms the pod passed its readinessProbe; the
# NodePort's iptables rules can lag slightly behind that, and it's this
# externally-reachable path (not the in-cluster probe) that journeys/CI
# actually depend on — so confirm it directly before wiring the Region CR.
wait_for_http_ok "${REGIONAL_API_ENDPOINT}/healthz" "regional" || exit 1

# 10. Generate and apply the Region CR to the global cluster
echo "--- Step 9: Registering regional cluster in the global cluster ---"
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
    - name: "seca.workspace"
      url: "${REGIONAL_API_ENDPOINT}/providers/seca.workspace"
      version: "v1"
    - name: "seca.storage"
      url: "${REGIONAL_API_ENDPOINT}/providers/seca.storage"
      version: "v1"
    - name: "seca.compute"
      url: "${REGIONAL_API_ENDPOINT}/providers/seca.compute"
      version: "v1"
    - name: "seca.network"
      url: "${REGIONAL_API_ENDPOINT}/providers/seca.network"
      version: "v1"
EOF

# 11. Discover the global service endpoint (both services are NodePort, so
# they're reachable directly at <control-plane-container-ip>:<nodePort> —
# no port-forward needed).
echo "--- Step 10: Discovering global service endpoint ---"
GLOBAL_NODE_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${GLOBAL_CLUSTER_NAME}-control-plane")
if [ -z "$GLOBAL_NODE_IP" ]; then
  echo "Error: Could not determine Node IP for the global cluster." >&2
  exit 1
fi

GLOBAL_NODE_PORT=$(kubectl --kubeconfig "${GLOBAL_KUBECONFIG_PATH}" get svc global-api-service -n default -o jsonpath='{.spec.ports[0].nodePort}')
if [ -z "$GLOBAL_NODE_PORT" ]; then
  echo "Error: Could not determine NodePort for global-api-service." >&2
  exit 1
fi

GLOBAL_API_ENDPOINT="http://${GLOBAL_NODE_IP}:${GLOBAL_NODE_PORT}"
echo "Constructed Global API Endpoint: ${GLOBAL_API_ENDPOINT}"

wait_for_http_ok "${GLOBAL_API_ENDPOINT}/healthz" "global" || exit 1

echo "--- Setup Complete! ---"
echo "To interact with the clusters, use:"
echo "export KUBECONFIG=${GLOBAL_KUBECONFIG_PATH}"
echo "kubectl get all"
echo ""
echo "export KUBECONFIG=${REGIONAL_KUBECONFIG_PATH}"
echo "kubectl get all"
echo ""
echo "Both APIs are exposed via NodePort on their control-plane container IP — reachable directly, no port-forward needed:"
echo "Global API:   ${GLOBAL_API_ENDPOINT}"
echo "Regional API: ${REGIONAL_API_ENDPOINT}"
echo ""
echo "For test/load journeys:"
echo "export BASE_URL_GLOBAL=${GLOBAL_API_ENDPOINT}"
echo "export BASE_URL_REGIONAL=${REGIONAL_API_ENDPOINT}"
echo ""
echo "To delete the clusters, run: kind delete cluster --name ${GLOBAL_CLUSTER_NAME} && kind delete cluster --name ${REGIONAL_CLUSTER_NAME}"