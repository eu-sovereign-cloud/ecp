#!/bin/bash

# prepare-generate-crd.sh
# This script prepares the environment for generating CRDs by patching Go types
# and ensuring the necessary imports are in place.
# Replaces time.Time with metav1.Time
# and handles map[string]interface{} to map[string]string conversion to avoid generation errors.

# Exit immediately if a command exits with a non-zero status.
set -e

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <api-file-path> <api-package-path> <crd-output-dir>"
    exit 1
fi

API_FILE_PATH="$1"
API_PACKAGE_PATH="$2"
CRD_OUTPUT_DIR="$3"

echo "Patching Go types for CRD generation in $API_FILE_PATH..."
# Replace time.Time with metav1.Time for proper CRD generation
sed -i 's/time\.Time/metav1.Time/g' "$API_FILE_PATH"

# Workaround - Replace map[string]interface{} with map[string]string to avoid generation errors
sed -i 's/*map\[string\]interface{}/\*map[string]string/g' "$API_FILE_PATH"

# Add metav1 import if it's not already there
sed -i 's|.*"time".*|	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"|' "$API_FILE_PATH"

echo "Generating CRD from $API_PACKAGE_PATH..."
# Clean up and create the output directory
rm -rf "$CRD_OUTPUT_DIR"
mkdir -p "$CRD_OUTPUT_DIR"

echo "CRD generation complete for $API_PACKAGE_PATH."