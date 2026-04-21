#!/usr/bin/env bash
# Verify the git working tree is clean (no uncommitted changes).
#
# Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation> [path...]
#   <repo-root>     absolute path to the repository root
#   <action>        human-readable name of the action that was just run
#                   (e.g. "workspace-sync", "generate-api")
#   <remediation>   command the developer should run to fix the issue
#                   (e.g. "make workspace-sync", "make generate-api")
#   [path...]       optional paths to scope the check (relative to repo root).
#                   When omitted the entire tree is checked.
#
# Exit code: 0 if clean, 1 if dirty

set -euo pipefail

repo_root="${1:?Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation> [path...]}"
action="${2:?Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation> [path...]}"
remediation="${3:?Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation> [path...]}"
shift 3

# Remaining arguments are paths to scope the check; empty means whole tree.
dirty=$(cd "${repo_root}" && git status --porcelain -- "$@")
if [ -n "${dirty}" ]; then
  echo "::error::${action} produced uncommitted changes in:"
  echo "${dirty}" | awk '{print "  " $2}'
  echo ""
  echo "  Run '${remediation}' and commit the results."
  exit 1
fi
