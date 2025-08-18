#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration (should match setup-dev-clusters.sh) ---
GLOBAL_CLUSTER_NAME="global"
REGIONAL_CLUSTER_NAME="regional"
KUBECONFIG_DIR="${HOME}/.kube/multi-cluster-demo"

# --- Helper Function ---
check_command() {
  if ! command -v "$1" &> /dev/null; then
    echo "Error: '$1' is not installed. Please install it to continue." >&2
    exit 1
  fi
}

# --- Script ---

# 1. Check for dependencies
check_command "kind"

echo "--- Starting cleanup of development clusters ---"

# 2. Delete kind clusters if they exist
echo "--- Step 1: Deleting 'global' and 'regional' kind clusters ---"

if kind get clusters | grep -q "^${GLOBAL_CLUSTER_NAME}$"; then
    echo "Deleting cluster '${GLOBAL_CLUSTER_NAME}'..."
    kind delete cluster --name "${GLOBAL_CLUSTER_NAME}"
else
    echo "Cluster '${GLOBAL_CLUSTER_NAME}' not found, skipping deletion."
fi

if kind get clusters | grep -q "^${REGIONAL_CLUSTER_NAME}$"; then
    echo "Deleting cluster '${REGIONAL_CLUSTER_NAME}'..."
    kind delete cluster --name "${REGIONAL_CLUSTER_NAME}"
else
    echo "Cluster '${REGIONAL_CLUSTER_NAME}' not found, skipping deletion."
fi

# 3. Remove the kubeconfig directory
if [ -d "${KUBECONFIG_DIR}" ]; then
    echo "--- Step 2: Removing kubeconfig directory '${KUBECONFIG_DIR}' ---"
    rm -rf "${KUBECONFIG_DIR}"
else
    echo "Kubeconfig directory '${KUBECONFIG_DIR}' not found, skipping removal."
fi

echo "--- Cleanup Complete! ---"