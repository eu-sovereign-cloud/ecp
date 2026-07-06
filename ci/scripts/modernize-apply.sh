#!/usr/bin/env bash
# Apply Go modernizations to a module, iterating to a fixpoint.
#
# Usage: modernize-apply.sh <module-dir>
#
# `go fix` analyzes the *original* source on each run, so one fix can expose
# another: e.g. the stditerators modernizer rewrites a counter loop into
# `for v := range s.Fields()` and leaves a now-redundant `v := v`, which only
# the forvar modernizer removes — on the *next* run. A single `go fix` pass
# therefore leaves partially-modernized code. We re-run until `go fix -diff`
# reports nothing pending (the fixpoint), bounded by MODERNIZE_MAX_PASSES as a
# guard against an analyzer that never settles.
#
# Exit code: 0 once modernized (or already so), 1 if it fails to converge.

set -euo pipefail

module_dir="${1:?Usage: modernize-apply.sh <module-dir>}"
max_passes="${MODERNIZE_MAX_PASSES:-10}"

cd "${module_dir}"

pass=0
while ! go fix -diff ./... >/dev/null 2>&1; do
  pass=$((pass + 1))
  if [ "${pass}" -gt "${max_passes}" ]; then
    echo "error: modernize did not converge after ${max_passes} passes" >&2
    go fix -diff ./... >&2 || true
    exit 1
  fi
  echo "  pass ${pass}: applying modernizations"
  go fix ./...
done

if [ "${pass}" -eq 0 ]; then
  echo "  already modernized"
fi
