#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
check_component_arg "$1"
source_config
setup_registry_vars "$1"

FULL_IMAGE_NAME=$IMAGE_NAME

# Generate the local tag for KIND
USE_KIND=true setup_registry_vars "$1"
LOCAL_IMAGE_NAME=$IMAGE_NAME

DOCKER_BUILD_CONTEXT="${SCRIPT_DIR}/../../../.."
DOCKERFILE_PATH="${SCRIPT_DIR}/../build/${COMPONENT}/Dockerfile"

# Build with the full name
docker build -t "${FULL_IMAGE_NAME}" -f "${DOCKERFILE_PATH}" "${DOCKER_BUILD_CONTEXT}"
echo "Image built: ${FULL_IMAGE_NAME}"

# Re-tag for local/KIND use if the names are different
if [ "${FULL_IMAGE_NAME}" != "${LOCAL_IMAGE_NAME}" ]; then
    docker tag "${FULL_IMAGE_NAME}" "${LOCAL_IMAGE_NAME}"
    echo "Image also tagged as: ${LOCAL_IMAGE_NAME}"
fi


