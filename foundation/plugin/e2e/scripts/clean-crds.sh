#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
setup_kube_vars

CRDS_DIR="${SCRIPT_DIR}/../../../api/generated/crds"

if [ -d "${CRDS_DIR}" ]; then
    echo "Cleaning CRDs from ${CRDS_DIR}..."
    find "${CRDS_DIR}" -type f -name "*.yaml" -exec cat {} + | kubectl ${KUBECONFIG_ARG} delete --ignore-not-found=true -f -
else
    echo "CRD directory not found, skipping cleanup."
fi
