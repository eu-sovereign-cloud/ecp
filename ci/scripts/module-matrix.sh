#!/usr/bin/env bash
# Resolve the pre-merge.yaml module check-matrix for a pull request.
#
# Usage: module-matrix.sh <builder-changed> <changed-modules-json>
#   builder-changed       'true' | 'false'  (dorny `builder` filter output)
#   changed-modules-json  JSON array of changed module filter keys
# Output: JSON array of modules for the matrix.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

builder_changed="${1:?Usage: module-matrix.sh <builder-changed> <changed-modules-json>}"
changed_modules="${2:?Usage: module-matrix.sh <builder-changed> <changed-modules-json>}"

if [ "${builder_changed}" = "true" ]; then
  "${script_dir}/go-modules.sh" --json
else
  printf '%s\n' "${changed_modules}"
fi
