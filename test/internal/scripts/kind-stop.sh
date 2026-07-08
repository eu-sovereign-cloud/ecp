#!/bin/bash
source "$(dirname "$0")/common.sh"

export USE_KIND=true
setup_kube_vars


echo "Deleting KIND cluster '${CLUSTER_NAME}'..."
kind delete cluster --name "${CLUSTER_NAME}"
echo "KIND cluster deleted."
