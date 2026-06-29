#!/usr/bin/env bash
# Check that a Go module is modernized via `go fix -diff`.
#
# Usage: modernize-check.sh <module-dir> <module-name>
#   <module-dir>    absolute path to the module directory
#   <module-name>   module name for error messages (e.g. "gateway")
#
# `go fix -diff` prints the unified diff of the modernizations it would apply
# and exits non-zero when that diff is non-empty — it never mutates the tree.
# This mirrors gofmt-check.sh: CI runs the check, the developer runs
# `make <module>-modernize` to apply the fixes.
#
# Exit code: 0 if already modernized, 1 if modernizations are pending

set -euo pipefail

module_dir="${1:?Usage: modernize-check.sh <module-dir> <module-name>}"
module_name="${2:?Usage: modernize-check.sh <module-dir> <module-name>}"

if ! (cd "${module_dir}" && go fix -diff ./...); then
  echo "FAIL: ${module_name} has un-modernized code (run: make ${module_name}-modernize)"
  exit 1
fi
