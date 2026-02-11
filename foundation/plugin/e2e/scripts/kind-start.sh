#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
export USE_KIND=true
setup_kube_vars

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

echo "Creating KIND cluster '${CLUSTER_NAME}'..."
kind create cluster --name "${CLUSTER_NAME}"

echo "KIND cluster '${CLUSTER_NAME}' started."

