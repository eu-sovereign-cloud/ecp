#!/bin/bash
set -e

# This script starts the regional gateway server.
# It uses environment variables for configuration and passes them as flags to the 'ecp' binary.

# Set default values if environment variables are not provided.
: "${REGIONAL_HOST:=0.0.0.0}"
: "${REGIONAL_PORT:=8080}"

# The kubeconfig default is handled by the binary itself, but we can pass it if set.
KUBECONFIG_FLAG=""
if [ -n "$KUBECONFIG" ]; then
  KUBECONFIG_FLAG="--kubeconfig=$KUBECONFIG"
fi

# We run the 'regionalapiserver' subcommand of the 'ecp' binary.
# We pass any extra arguments ($@) to the binary.
echo "Starting regional gateway..."
./ecp regionalapiserver --regionalHost="$REGIONAL_HOST" --regionalPort="$REGIONAL_PORT" $KUBECONFIG_FLAG "$@"
