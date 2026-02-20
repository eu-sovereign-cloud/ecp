#!/bin/bash

set -euo pipefail

# --- Discover paths ---
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
GATEWAY_ROOT="${SCRIPT_DIR}/.."
# Determine repository root (git preferred)
REPO_ROOT="$(git -C "${GATEWAY_ROOT}" rev-parse --show-toplevel 2>/dev/null || echo "${GATEWAY_ROOT}/../..")"
BUILD_DIR="${GATEWAY_ROOT}/build"

if [ ! -d "${BUILD_DIR}" ]; then
  echo "ERROR: build directory not found at ${BUILD_DIR}" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker CLI not found in PATH" >&2
  exit 2
fi

# --- Configuration (override via env) ---
REGISTRY="${REGISTRY:-registry.secapi.cloud}"
GLOBAL_IMAGE_TAG="${GLOBAL_IMAGE_TAG:-global-server:latest}"
REGIONAL_IMAGE_TAG="${REGIONAL_IMAGE_TAG:-regional-server:latest}"

GLOBAL_IMAGE_REF="${REGISTRY}/${GLOBAL_IMAGE_TAG}"
REGIONAL_IMAGE_REF="${REGISTRY}/${REGIONAL_IMAGE_TAG}"

# --- Script ---

echo "--- Building Docker images (context: ${REPO_ROOT}) ---"

echo "Building global server image: ${GLOBAL_IMAGE_REF}"
docker build -t "${GLOBAL_IMAGE_REF}" -f "${BUILD_DIR}/Dockerfile.global" "${REPO_ROOT}"

echo "Building regional server image: ${REGIONAL_IMAGE_REF}"
docker build -t "${REGIONAL_IMAGE_REF}" -f "${BUILD_DIR}/Dockerfile.regional" "${REPO_ROOT}"

echo "--- Docker images built successfully! ---"