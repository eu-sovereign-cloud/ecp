#!/bin/bash
set -e

# This script starts the global gateway server.
# It uses environment variables for configuration and passes them as flags to the 'ecp' binary.

# Set default values if environment variables are not provided.
: "${GLOBAL_HOST:=0.0.0.0}"
: "${GLOBAL_PORT:=8080}"

# The kubeconfig default is handled by the binary itself, but we can pass it if set.
KUBECONFIG_FLAG=""
if [ -n "$KUBECONFIG" ]; then
  KUBECONFIG_FLAG="--kubeconfig=$KUBECONFIG"
fi

# The 'ecp' binary runs the global server by default.
# We pass any extra arguments ($@) to the binary.
echo "Starting global gateway..."
./ecp globalapiserver --host="$GLOBAL_HOST" --port="$GLOBAL_PORT" $KUBECONFIG_FLAG "$@"
