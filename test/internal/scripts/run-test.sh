#!/bin/bash
set -e

COMPONENT=$1

if [ -z "$COMPONENT" ]; then
    echo "Error: Component name must be provided as the first argument." >&2
    exit 1
fi
shift # consume the component; forward any remaining args to `go test` (e.g. -run=TestBench)

echo "--- Running Integration Tests for ${COMPONENT} ---"

source "$(dirname "$0")/common.sh"
setup_env
setup_kube_vars

go test -v -count=1 -tags=integration,authhelper -timeout "${TEST_TIMEOUT:-20m}" ./integration/${COMPONENT}/...
