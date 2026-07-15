#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"
source_config

export USE_KIND=true
setup_kube_vars
setup_registry_vars "$1"

echo "Loading image ${IMAGE_NAME} into KIND cluster '${CLUSTER_NAME}'..."
kind load docker-image "${IMAGE_NAME}" --name "${CLUSTER_NAME}"

