#!/usr/bin/env bash
# Verify the git working tree is clean (no uncommitted changes).
#
# Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation>
#   <repo-root>     absolute path to the repository root
#   <action>        human-readable name of the action that was just run
#                   (e.g. "workspace-sync", "generate-api")
#   <remediation>   command the developer should run to fix the issue
#                   (e.g. "make workspace-sync", "make generate-api")
#
# Exit code: 0 if tree is clean, 1 if dirty

set -euo pipefail

repo_root="${1:?Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation>}"
action="${2:?Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation>}"
remediation="${3:?Usage: git-tree-clean-verify.sh <repo-root> <action> <remediation>}"

dirty=$(cd "${repo_root}" && git status --porcelain)
if [ -n "${dirty}" ]; then
  echo "::error::${action} produced uncommitted changes in:"
  echo "${dirty}" | awk '{print "  " $2}'
  echo ""
  echo "  Run '${remediation}' and commit the results."
  exit 1
fi
