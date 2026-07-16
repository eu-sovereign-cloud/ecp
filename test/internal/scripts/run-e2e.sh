#!/bin/bash
# Run the single end-to-end suite against a cluster that already has test-data,
# both gateways and the delegator deployed. Honours the same context/kubeconfig
# resolution as the other scripts (KIND, context/kubeconfig.yaml, or ambient).
set -e

source "$(dirname "$0")/common.sh"
setup_env
setup_kube_vars

echo "--- Running end-to-end suite ---"
go test -v -count=1 -tags=e2e,authhelper -timeout "${TEST_TIMEOUT:-20m}" ./e2e/...
