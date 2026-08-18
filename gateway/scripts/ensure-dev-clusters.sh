#!/usr/bin/env bash
# Idempotent bootstrap for the 'global'/'regional' KIND dev clusters: if both
# already exist, no-op; otherwise build the server images and run the ionos
# conformance suite's setup-clusters.sh. Used by test/load's *-dev Make targets
# so they work as a single command on a machine with no dev clusters yet.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

existing="$(kind get clusters 2>/dev/null || true)"
if grep -qx "global" <<<"${existing}" && grep -qx "regional" <<<"${existing}"; then
	echo "ensure-dev-clusters: 'global' and 'regional' KIND clusters already exist"
	exit 0
fi

echo "ensure-dev-clusters: 'global'/'regional' dev clusters missing — bootstrapping (this takes a few minutes)"
"${SCRIPT_DIR}/build-images.sh"
"${REPO_ROOT}/test/conformance/ionos/scripts/setup-clusters.sh"
