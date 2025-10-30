#!/bin/bash

set -euo pipefail

# --- Configuration ---
GLOBAL_IMAGE="global-server:latest"
REGIONAL_IMAGE="regional-server:latest"

# --- Script ---

echo "--- Building Docker images ---"

echo "Building global server image: ${GLOBAL_IMAGE}"
docker build -t "${GLOBAL_IMAGE}" -f build/Dockerfile.global .

echo "Building regional server image: ${REGIONAL_IMAGE}"
docker build -t "${REGIONAL_IMAGE}" -f build/Dockerfile.regional .

echo "--- Docker images built successfully! ---"