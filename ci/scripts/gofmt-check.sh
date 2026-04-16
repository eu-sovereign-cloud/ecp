#!/usr/bin/env bash
# Check formatting of a Go module via golangci-lint fmt --diff.
#
# Usage: gofmt-check.sh <module-dir> <module-name>
#   <module-dir>    absolute path to the module directory
#   <module-name>   module name for error messages (e.g. "foundation/gateway")
#
# Exit code: 0 if formatted, 1 if unformatted files found

set -euo pipefail

module_dir="${1:?Usage: gofmt-check.sh <module-dir> <module-name>}"
module_name="${2:?Usage: gofmt-check.sh <module-dir> <module-name>}"

diff=$(cd "${module_dir}" && golangci-lint fmt --diff ./... 2>&1) || true

if [ -n "${diff}" ]; then
  printf '%s\n' "${diff}"
  echo "FAIL: ${module_name} has unformatted files (run: make ${module_name}-gofmt)"
  exit 1
fi
