#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"

setup_kube_vars
DEPLOY_DIR="${SCRIPT_DIR}/../deploy/${COMPONENT}"
SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-e2e-ecp}"

echo "Deleting ${COMPONENT} from Kubernetes cluster..."

if setup_chart_vars "${COMPONENT}"; then
    helm ${HELM_KUBECONFIG_ARG} uninstall "${HELM_RELEASE}" \
        --namespace "${SYSTEM_NAMESPACE}" --ignore-not-found --wait
elif command -v kubectl &> /dev/null && kubectl kustomize --help > /dev/null 2>&1; then
    kubectl kustomize "${DEPLOY_DIR}" | kubectl ${KUBECONFIG_ARG} delete --ignore-not-found=true -f -
else
    echo "kubectl kustomize is not available, using kubectl delete -k. "
    kubectl ${KUBECONFIG_ARG} delete --ignore-not-found=true -k "${DEPLOY_DIR}"
fi

echo "Cleanup of ${COMPONENT} complete."

