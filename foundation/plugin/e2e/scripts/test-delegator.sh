#!/bin/bash
source "$(dirname "$0")/common.sh"

setup_env
setup_kube_vars

echo "Running delegator integration tests..."
go test -v -count=1 -tags=integration ./test/integration/delegator/...
