#!/bin/bash
set -eo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

REGISTRY=${REGISTRY:-""}
IMG=${IMG:-"ecp-dummy-delegator"}
VERSION=${VERSION:-"latest"}

KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
export KUBECONFIG

CRDS_DIR="${SCRIPT_DIR}/../../../api/generated/crds"
DEPLOY_DIR="${SCRIPT_DIR}/../deploy"

# Set image name for kustomize
if [ -n "$REGISTRY" ]; then
    IMAGE_NAME="${REGISTRY}/${IMG}:${VERSION}"
else
    IMAGE_NAME="${IMG}:${VERSION}"
fi

echo "Applying CRDs from ${CRDS_DIR}..."
find "${CRDS_DIR}" -type f -name "*.yaml" -exec cat {} + | kubectl apply -f -

if command -v kustomize &> /dev/null; then
    echo "Setting image in kustomization to ${IMAGE_NAME}..."
    (cd "${DEPLOY_DIR}" && kustomize edit set image "dummy-delegator-image=${IMAGE_NAME}")

    echo "Deploying dummy delegator via kustomize..."
    kustomize build "${DEPLOY_DIR}" | kubectl apply -f -
else
    echo "kustomize not found, deploying via kubectl apply -k. The image must be correctly set in kustomization.yaml"
    kubectl apply -k "${DEPLOY_DIR}"
fi

echo "Deployment complete."
