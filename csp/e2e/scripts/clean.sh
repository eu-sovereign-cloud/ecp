#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"

setup_kube_vars
DEPLOY_DIR="${SCRIPT_DIR}/../deploy/${COMPONENT}"

echo "Deleting ${COMPONENT} from Kubernetes cluster..."

if command -v kubectl &> /dev/null && kubectl kustomize --help > /dev/null 2>&1; then
    kubectl kustomize "${DEPLOY_DIR}" | kubectl ${KUBECONFIG_ARG} delete --ignore-not-found=true -f -
else
    echo "kubectl kustomize is not available, using kubectl delete -k. "
    kubectl ${KUBECONFIG_ARG} delete --ignore-not-found=true -k "${DEPLOY_DIR}"
fi

echo "Cleanup of ${COMPONENT} complete."

