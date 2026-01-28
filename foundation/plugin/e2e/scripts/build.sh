#!/bin/bash
set -eo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

IMG=${IMG:-"ecp-e2e-delegator"}
VERSION=${VERSION:-"latest"}

if [ -n "$REGISTRY" ]; then
    IMAGE_NAME="${REGISTRY}/${IMG}:${VERSION}"
else
    IMAGE_NAME="${IMG}:${VERSION}"
fi


DOCKER_BUILD_CONTEXT="${SCRIPT_DIR}/../../../.."
DOCKERFILE_PATH="${SCRIPT_DIR}/../build/Dockerfile"

docker build -t "${IMAGE_NAME}" -f "${DOCKERFILE_PATH}" "${DOCKER_BUILD_CONTEXT}"

echo "Image built: ${IMAGE_NAME}"
