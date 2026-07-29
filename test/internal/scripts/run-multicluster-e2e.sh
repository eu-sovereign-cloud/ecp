#!/bin/bash
# Run the multicluster end-to-end suite against a pair of clusters that already
# have the split stack deployed and the region registered (see
# `make kind-multicluster-stack`).
#
# Unlike the other runners this does NOT call setup_kube_vars: the suite talks to
# two clusters at once, so it takes explicit kubeconfig contexts rather than an
# ambient current-context.
set -e

source "$(dirname "$0")/common.sh"

setup_env
source_config

export MULTICLUSTER_GLOBAL_CONTEXT="${MULTICLUSTER_GLOBAL_CONTEXT:-kind-e2e-global}"
export MULTICLUSTER_REGIONAL_CONTEXT="${MULTICLUSTER_REGIONAL_CONTEXT:-kind-e2e-regional}"

echo "--- Running multicluster end-to-end suite ---"
echo "global:   ${MULTICLUSTER_GLOBAL_CONTEXT}"
echo "regional: ${MULTICLUSTER_REGIONAL_CONTEXT}"
go test -v -count=1 -tags=multicluster,authhelper -timeout "${TEST_TIMEOUT:-20m}" ./e2e/multicluster/...
