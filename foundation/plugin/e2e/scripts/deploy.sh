#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"
source_config

setup_kube_vars
setup_registry_vars "$1"

DEPLOY_DIR="${SCRIPT_DIR}/../deploy/${COMPONENT}"
CRDS_DIR="${SCRIPT_DIR}/../../../api/generated/crds"

echo "Applying CRDs from ${CRDS_DIR}..."
find "${CRDS_DIR}" -type f -name "*.yaml" -exec cat {} + | kubectl ${KUBECONFIG_ARG} apply -f -

# The default image name that 'kubectl kustomize' will output based on the kustomization.yaml files
DEFAULT_KUSTOMIZE_IMAGE="localhost/e2e-ecp-${COMPONENT}:latest"

echo "Deploying ${COMPONENT} with image ${IMAGE_NAME}..."

kubectl kustomize "${DEPLOY_DIR}" | \
    sed "s|${DEFAULT_KUSTOMIZE_IMAGE}|${IMAGE_NAME}|g" | \
    kubectl ${KUBECONFIG_ARG} apply -f -


echo "Deployment of ${COMPONENT} complete."

