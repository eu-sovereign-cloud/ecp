#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"
source_config
setup_registry_vars "$1"

echo "Pushing ${IMAGE_NAME}"

if [ -n "$REGISTRY_USER" ] && [ -n "$REGISTRY_PASSWORD" ]; then
    echo "${REGISTRY_PASSWORD}" | docker login "${REGISTRY_URL}" -u "${REGISTRY_USER}" --password-stdin
else
    echo "REGISTRY_USER and REGISTRY_PASSWORD are not set. Assuming you are already logged in."
fi

docker push "${IMAGE_NAME}"

echo "Image pushed: ${IMAGE_NAME}"

