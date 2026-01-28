#!/bin/bash
set -eo pipefail

CLUSTER_NAME=${CLUSTER_NAME:-"dummy-delegator-cluster"}

echo "Deleting KIND cluster '${CLUSTER_NAME}'..."
kind delete cluster --name "${CLUSTER_NAME}"
echo "KIND cluster deleted."
