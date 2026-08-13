#!/usr/bin/env bash
# Idempotent bootstrap for the 'global'/'regional' KIND dev clusters: if both
# already exist, no-op; otherwise build the server images and run
# setup-dev-clusters.sh. Used by test/load's *-dev Make targets so they work
# as a single command on a machine that hasn't run create-dev-clusters yet.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

existing="$(kind get clusters 2>/dev/null || true)"
if grep -qx "global" <<<"${existing}" && grep -qx "regional" <<<"${existing}"; then
	echo "ensure-dev-clusters: 'global' and 'regional' KIND clusters already exist"
	exit 0
fi

echo "ensure-dev-clusters: 'global'/'regional' dev clusters missing — bootstrapping (this takes a few minutes)"
"${SCRIPT_DIR}/build-images.sh"
"${SCRIPT_DIR}/setup-dev-clusters.sh"
