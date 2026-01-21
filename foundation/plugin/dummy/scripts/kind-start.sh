#!/bin/bash
set -eo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

export IMG=${IMG:-"ecp-dummy-delegator"}
export VERSION=${VERSION:-"latest"}
# For local development with kind, using 'localhost' as the registry
# helps ensure the image name is resolved locally.
export REGISTRY="localhost" 
CLUSTER_NAME=${CLUSTER_NAME:-"dummy-delegator-cluster"}

# Check if kind is installed
if ! command -v kind &> /dev/null
then
    echo "kind could not be found, please install it"
    exit 1
fi

if ! command -v kubectl &> /dev/null
then
    echo "kubectl could not be found, please install it"
    exit 1
fi

# The kustomize binary is optional, deploy.sh will use 'kubectl -k' as a fallback.


echo "Creating KIND cluster '${CLUSTER_NAME}'..."
kind create cluster --name "${CLUSTER_NAME}"

echo "Building image..."
/bin/bash "${SCRIPT_DIR}/build.sh"

echo "Loading image into KIND cluster..."
kind load docker-image "${REGISTRY}/${IMG}:${VERSION}" --name "${CLUSTER_NAME}"

echo "Deploying to KIND cluster..."
/bin/bash "${SCRIPT_DIR}/deploy.sh"

echo "KIND cluster started and delegator deployed."
